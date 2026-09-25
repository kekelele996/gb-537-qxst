package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"pki-certificate-rollover-impact/backend/internal/algorithm"
	"pki-certificate-rollover-impact/backend/internal/config"
	"pki-certificate-rollover-impact/backend/internal/constants"
	"pki-certificate-rollover-impact/backend/internal/model"
	"pki-certificate-rollover-impact/backend/internal/util"
	"pki-certificate-rollover-impact/backend/internal/x509util"
)

func Open(cfg config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch cfg.DBDriver {
	case "postgres":
		dialector = postgres.Open(cfg.DBDSN)
	case "sqlite":
		dialector = sqlite.Open(cfg.DBDSN)
	default:
		return nil, fmt.Errorf("unsupported database driver %q", cfg.DBDriver)
	}
	db, err := gorm.Open(dialector, &gorm.Config{Logger: logger.Default.LogMode(logger.Warn), TranslateError: true})
	if err != nil {
		return nil, fmt.Errorf("open %s database: %w", cfg.DBDriver, err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("obtain database connection pool: %w", err)
	}
	if cfg.DBDriver == "sqlite" {
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
	} else {
		sqlDB.SetMaxOpenConns(20)
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetConnMaxLifetime(30 * time.Minute)
		sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.TrustAnchor{}, &model.CertificateChain{}, &model.DependentService{}, &model.RolloverScenario{}, &model.AuditLog{}); err != nil {
		return nil, fmt.Errorf("migrate database schema: %w", err)
	}
	if err := seed(db); err != nil {
		return nil, err
	}
	return db, nil
}
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

type seedAccount struct {
	Username, DisplayName, Team, Password string
	Role                                  constants.Role
}

func seed(db *gorm.DB) error {
	accounts := []seedAccount{{"admin", "PKI System Administrator", "PKI Platform", "admin123", constants.RoleAdmin}, {"operator", "PKI Operations Lead", "PKI Platform", "operator123", constants.RolePKIOperator}, {"owner", "Payments Service Owner", "Payments Platform", "owner123", constants.RoleServiceOwner}, {"reviewer", "Independent Security Reviewer", "Security Assurance", "reviewer123", constants.RoleSecurityReviewer}, {"auditor", "Compliance Auditor", "Internal Audit", "auditor123", constants.RoleAuditor}}
	users := map[string]model.User{}
	for _, account := range accounts {
		var user model.User
		err := db.Where("username = ?", account.Username).First(&user).Error
		if err == nil {
			users[account.Username] = user
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("find seed user: %w", err)
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(account.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash seed password: %w", err)
		}
		now := time.Now().UTC()
		user = model.User{Username: account.Username, DisplayName: account.DisplayName, Team: account.Team, PasswordHash: string(hash), Role: string(account.Role), Active: true, CreatedAt: now, UpdatedAt: now}
		if err := db.Create(&user).Error; err != nil {
			return fmt.Errorf("create seed user: %w", err)
		}
		users[account.Username] = user
	}
	var count int64
	if err := db.Model(&model.TrustAnchor{}).Count(&count).Error; err != nil {
		return fmt.Errorf("count seed anchors: %w", err)
	}
	if count > 0 {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error { return seedDomain(tx, users) })
}

func seedDomain(tx *gorm.DB, users map[string]model.User) error {
	now := time.Now().UTC().Truncate(time.Second)
	pki, err := x509util.GenerateDemoPKI(now)
	if err != nil {
		return fmt.Errorf("generate demo public-key infrastructure: %w", err)
	}
	oldCertificate, err := x509util.ParseCertificatePEM(pki.OldRootPEM)
	if err != nil {
		return err
	}
	newCertificate, err := x509util.ParseCertificatePEM(pki.NewRootPEM)
	if err != nil {
		return err
	}
	anchors := []model.TrustAnchor{{AnchorCode: "ROOT-LEGACY-2022", SubjectDN: x509util.SubjectDN(oldCertificate), SerialNumber: oldCertificate.SerialNumber.String(), FingerprintSHA256: x509util.Fingerprint(oldCertificate), NotBefore: oldCertificate.NotBefore, NotAfter: oldCertificate.NotAfter, KeyAlgorithm: x509util.KeyAlgorithm(oldCertificate), CertificateState: string(constants.CalculateCertificateState(now, oldCertificate.NotAfter, false)), PemRedacted: x509util.NormalizePEM(oldCertificate), CreatedAt: now.Add(-180 * 24 * time.Hour), UpdatedAt: now}, {AnchorCode: "ROOT-NEXT-2031", SubjectDN: x509util.SubjectDN(newCertificate), SerialNumber: newCertificate.SerialNumber.String(), FingerprintSHA256: x509util.Fingerprint(newCertificate), NotBefore: newCertificate.NotBefore, NotAfter: newCertificate.NotAfter, KeyAlgorithm: x509util.KeyAlgorithm(newCertificate), CertificateState: string(constants.CalculateCertificateState(now, newCertificate.NotAfter, false)), PemRedacted: x509util.NormalizePEM(newCertificate), CreatedAt: now.Add(-7 * 24 * time.Hour), UpdatedAt: now}}
	if err := tx.Create(&anchors).Error; err != nil {
		return fmt.Errorf("create seed trust anchors: %w", err)
	}
	oldChain, err := seedChain("PLATFORM-TLS-LEGACY", anchors[0], pki.OldLeafPEM, now)
	if err != nil {
		return err
	}
	newChain, err := seedChain("PLATFORM-TLS-NEXT", anchors[1], pki.NewLeafPEM, now)
	if err != nil {
		return err
	}
	chains := []model.CertificateChain{oldChain, newChain}
	if err := tx.Create(&chains).Error; err != nil {
		return fmt.Errorf("create seed certificate chains: %w", err)
	}
	both, _ := json.Marshal([]uint{anchors[0].ID, anchors[1].ID})
	oldOnly, _ := json.Marshal([]uint{anchors[0].ID})
	none := `[]`
	services := []model.DependentService{{ServiceCode: "EDGE-AUTH", Name: "Edge Authentication Gateway", OwnerTeam: "PKI Platform", Environment: "production", ChainID: chains[0].ID, ClientTrustRefsJSON: string(both), Protocol: "mtls", Criticality: "critical", DependencyEdgesJSON: none, ServiceState: string(constants.ServiceActive), CreatedAt: now.Add(-90 * 24 * time.Hour), UpdatedAt: now}, {ServiceCode: "PAYMENTS-API", Name: "Payments Transaction API", OwnerTeam: "Payments Platform", Environment: "production", ChainID: chains[0].ID, ClientTrustRefsJSON: string(oldOnly), Protocol: "mtls", Criticality: "critical", DependencyEdgesJSON: none, ServiceState: string(constants.ServiceActive), CreatedAt: now.Add(-60 * 24 * time.Hour), UpdatedAt: now}, {ServiceCode: "ORDER-WORKER", Name: "Order Settlement Worker", OwnerTeam: "Payments Platform", Environment: "production", ChainID: chains[0].ID, ClientTrustRefsJSON: string(both), Protocol: "kafka_tls", Criticality: "high", DependencyEdgesJSON: none, ServiceState: string(constants.ServiceActive), CreatedAt: now.Add(-45 * 24 * time.Hour), UpdatedAt: now}}
	if err := tx.Create(&services).Error; err != nil {
		return fmt.Errorf("create seed dependent services: %w", err)
	}
	paymentsEdge, _ := json.Marshal([]uint{services[0].ID})
	workerEdge, _ := json.Marshal([]uint{services[1].ID})
	if err := tx.Model(&services[1]).Update("dependency_edges_json", string(paymentsEdge)).Error; err != nil {
		return err
	}
	services[1].DependencyEdgesJSON = string(paymentsEdge)
	if err := tx.Model(&services[2]).Update("dependency_edges_json", string(workerEdge)).Error; err != nil {
		return err
	}
	services[2].DependencyEdgesJSON = string(workerEdge)
	snapshot := algorithm.NewSnapshot(algorithm.ScenarioConfig{Name: "Production platform root rollover rehearsal", OldAnchorID: anchors[0].ID, NewAnchorID: anchors[1].ID, OverlapStart: now.Add(14 * 24 * time.Hour), OverlapEnd: now.Add(30 * 24 * time.Hour), CandidateChainIDs: []uint{chains[0].ID, chains[1].ID}, SimulationTime: now.Add(21 * 24 * time.Hour)}, []algorithm.AnchorSnapshot{{ID: anchors[0].ID, Code: anchors[0].AnchorCode, State: anchors[0].CertificateState, NotBefore: anchors[0].NotBefore, NotAfter: anchors[0].NotAfter}, {ID: anchors[1].ID, Code: anchors[1].AnchorCode, State: anchors[1].CertificateState, NotBefore: anchors[1].NotBefore, NotAfter: anchors[1].NotAfter}}, []algorithm.ChainSnapshot{{ID: chains[0].ID, Code: chains[0].ChainCode, AnchorID: chains[0].TrustAnchorID, LeafSubject: chains[0].LeafSubject, ValidFrom: chains[0].ValidFrom, ValidTo: chains[0].ValidTo, State: chains[0].ChainState, ValidationValid: true}, {ID: chains[1].ID, Code: chains[1].ChainCode, AnchorID: chains[1].TrustAnchorID, LeafSubject: chains[1].LeafSubject, ValidFrom: chains[1].ValidFrom, ValidTo: chains[1].ValidTo, State: chains[1].ChainState, ValidationValid: true}}, []algorithm.ServiceSnapshot{{ID: services[0].ID, Code: services[0].ServiceCode, ChainID: services[0].ChainID, TrustAnchorIDs: []uint{anchors[0].ID, anchors[1].ID}, Criticality: services[0].Criticality, State: services[0].ServiceState}, {ID: services[1].ID, Code: services[1].ServiceCode, ChainID: services[1].ChainID, TrustAnchorIDs: []uint{anchors[0].ID}, DependencyIDs: []uint{services[0].ID}, Criticality: services[1].Criticality, State: services[1].ServiceState}, {ID: services[2].ID, Code: services[2].ServiceCode, ChainID: services[2].ChainID, TrustAnchorIDs: []uint{anchors[0].ID, anchors[1].ID}, DependencyIDs: []uint{services[1].ID}, Criticality: services[2].Criticality, State: services[2].ServiceState}})
	result, err := algorithm.Simulate(snapshot)
	if err != nil {
		return fmt.Errorf("simulate seed rollover: %w", err)
	}
	hash, err := snapshot.Hash()
	if err != nil {
		return err
	}
	snapshotJSON, _ := snapshot.Canonical()
	candidateJSON, _ := json.Marshal(snapshot.Config.CandidateChainIDs)
	affectedJSON, _ := json.Marshal(result.AffectedServices)
	pathsJSON, _ := json.Marshal(result.BrokenPaths)
	evidenceJSON, _ := json.Marshal(result.Evidence)
	operator := users["operator"]
	scenario := model.RolloverScenario{Name: snapshot.Config.Name, OldAnchorID: anchors[0].ID, NewAnchorID: anchors[1].ID, OverlapStart: snapshot.Config.OverlapStart, OverlapEnd: snapshot.Config.OverlapEnd, CandidateChainIDs: string(candidateJSON), AlgorithmVersion: algorithm.Version, InputHash: hash, InputSnapshot: snapshotJSON, SimulationTime: snapshot.Config.SimulationTime, AffectedServicesJSON: string(affectedJSON), BrokenPathsJSON: string(pathsJSON), PathEvidenceJSON: string(evidenceJSON), ScenarioState: string(constants.ScenarioSimulated), Explanation: result.Explanation, CreatedBy: operator.ID, CreatedByName: operator.Username, IdempotencyKey: "seed-rollover-simulation", DurationMS: 1, CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)}
	if err := tx.Create(&scenario).Error; err != nil {
		return fmt.Errorf("create seed rollover scenario: %w", err)
	}
	return nil
}

func seedChain(code string, anchor model.TrustAnchor, leafPEM string, now time.Time) (model.CertificateChain, error) {
	evidence, refs, err := x509util.ValidateChain(anchor.PemRedacted, leafPEM, now)
	if err != nil {
		return model.CertificateChain{}, fmt.Errorf("validate seed chain: %w", err)
	}
	certificates, err := x509util.ParseCertificateBundle(leafPEM)
	if err != nil {
		return model.CertificateChain{}, err
	}
	refsJSON, _ := json.Marshal(refs)
	evidenceJSON, _ := json.Marshal(evidence)
	fingerprints := []string{}
	for _, ref := range refs {
		fingerprints = append(fingerprints, ref.FingerprintSHA256)
	}
	validFrom := certificates[0].NotBefore
	if anchor.NotBefore.After(validFrom) {
		validFrom = anchor.NotBefore
	}
	validTo := certificates[0].NotAfter
	if anchor.NotAfter.Before(validTo) {
		validTo = anchor.NotAfter
	}
	return model.CertificateChain{ChainCode: code, TrustAnchorID: anchor.ID, LeafSubject: certificates[0].Subject.String(), CertificateRefsJSON: string(refsJSON), ChainFingerprint: util.HashString(fmt.Sprint(fingerprints)), ValidFrom: validFrom, ValidTo: validTo, ValidationResult: string(evidenceJSON), ChainState: string(constants.ChainValidated), SourceChecksum: util.HashString(leafPEM), PublicChainPEM: leafPEM, CreatedAt: now, UpdatedAt: now}, nil
}
