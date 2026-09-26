package service

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http"
	"pki-certificate-rollover-impact/backend/internal/algorithm"
	"pki-certificate-rollover-impact/backend/internal/constants"
	"pki-certificate-rollover-impact/backend/internal/dto"
	"pki-certificate-rollover-impact/backend/internal/model"
	"pki-certificate-rollover-impact/backend/internal/repository"
	"pki-certificate-rollover-impact/backend/internal/util"
	"testing"
	"time"
)

func TestBusinessWriteRollsBackWhenAuditStepFails(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:transaction-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.TrustAnchor{}); err != nil {
		t.Fatal(err)
	}
	manager := repository.NewTransactionManager(db)
	anchors := repository.NewTrustAnchorRepository(db)
	expected := errors.New("audit append failed")
	err = manager.WithinTransaction(context.Background(), func(ctx context.Context) error {
		anchor := model.TrustAnchor{AnchorCode: "ROLLBACK", SubjectDN: "CN=rollback", SerialNumber: "1", FingerprintSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", KeyAlgorithm: "ECDSA", CertificateState: "valid", PemRedacted: "certificate"}
		if err := anchors.Create(ctx, &anchor); err != nil {
			return err
		}
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("got %v want %v", err, expected)
	}
	var count int64
	if err := db.Model(&model.TrustAnchor{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("business write committed without audit: %d", count)
	}
}

func TestOwnershipAndReviewerSeparation(t *testing.T) {
	owner := util.Actor{UserID: 3, Role: string(constants.RoleServiceOwner), Team: "Payments"}
	if err := requireServiceOwnership(owner, "Payments"); err != nil {
		t.Fatal(err)
	}
	if err := requireServiceOwnership(owner, "Identity"); err == nil {
		t.Fatal("cross-team update should be denied")
	}
	scenario := model.RolloverScenario{CreatedBy: 3}
	if !scenario.ReviewerSeparated(4) || scenario.ReviewerSeparated(3) {
		t.Fatal("reviewer separation invariant failed")
	}
}

func TestScenarioCreatorCannotVerifyOwnSimulation(t *testing.T) {
	db := newScenarioTestDB(t)
	scenario := persistScenario(t, db, minimalScenario(t, db, "separation", "separation-hash", "", "executing", 7, 0))
	service := NewRolloverScenarioService(repository.NewRolloverScenarioRepository(db), nil, nil, nil, repository.NewAuditRepository(db), repository.NewTransactionManager(db))

	_, err := service.Transition(context.Background(), scenario.ID, dto.RolloverScenarioTransitionRequest{ToState: string(constants.ScenarioVerified)}, util.Actor{UserID: 7, Username: "operator", Role: string(constants.RolePKIOperator)}, "request-self-verify")
	var apiErr *util.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusConflict || apiErr.Code != util.CodeReviewerConflict {
		t.Fatalf("got %#v, want 409 %s", err, util.CodeReviewerConflict)
	}
}

type failingAuditRepository struct{ err error }

func (r failingAuditRepository) Record(context.Context, model.AuditLog) error { return r.err }
func (r failingAuditRepository) List(context.Context, repository.AuditQuery) ([]model.AuditLog, int64, error) {
	return nil, 0, r.err
}

func newScenarioTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.TrustAnchor{}, &model.RolloverScenario{}, &model.AuditLog{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func persistScenario(t *testing.T, db *gorm.DB, scenario model.RolloverScenario) model.RolloverScenario {
	t.Helper()
	if err := db.Create(&scenario).Error; err != nil {
		t.Fatal(err)
	}
	return scenario
}

func minimalScenario(t *testing.T, db *gorm.DB, name, inputHash, key, state string, createdBy uint, offset time.Duration) model.RolloverScenario {
	t.Helper()
	now := time.Date(2032, 4, 2, 8, 0, 0, 0, time.UTC).Add(offset)
	var anchors []model.TrustAnchor
	if err := db.Order("id ASC").Find(&anchors).Error; err != nil {
		t.Fatal(err)
	}
	if len(anchors) < 2 {
		anchors = []model.TrustAnchor{
			{AnchorCode: "TEST-OLD", SubjectDN: "CN=old", SerialNumber: "1", FingerprintSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", NotBefore: now.Add(-24 * time.Hour), NotAfter: now.Add(365 * 24 * time.Hour), KeyAlgorithm: "ECDSA", CertificateState: "valid", PemRedacted: "public certificate", CreatedAt: now, UpdatedAt: now},
			{AnchorCode: "TEST-NEW", SubjectDN: "CN=new", SerialNumber: "2", FingerprintSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", NotBefore: now.Add(-24 * time.Hour), NotAfter: now.Add(365 * 24 * time.Hour), KeyAlgorithm: "ECDSA", CertificateState: "valid", PemRedacted: "public certificate", CreatedAt: now, UpdatedAt: now},
		}
		if err := db.Create(&anchors).Error; err != nil {
			t.Fatal(err)
		}
	}
	return model.RolloverScenario{Name: name, OldAnchorID: anchors[0].ID, NewAnchorID: anchors[1].ID, OverlapStart: now.Add(time.Hour), OverlapEnd: now.Add(2 * time.Hour), CandidateChainIDs: "[]", AlgorithmVersion: algorithm.Version, InputHash: inputHash, InputSnapshot: "{}", SimulationTime: now.Add(90 * time.Minute), AffectedServicesJSON: "[]", BrokenPathsJSON: "[]", PathEvidenceJSON: "[]", ScenarioState: state, Explanation: "test", CreatedBy: createdBy, CreatedByName: "operator", IdempotencyKey: key, CreatedAt: now, UpdatedAt: now}
}

func TestSimulationIdempotencyKeyCannotCrossScenarioBoundary(t *testing.T) {
	db := newScenarioTestDB(t)
	first := persistScenario(t, db, minimalScenario(t, db, "first", "hash-first", "shared-key", "simulated", 7, 0))
	second := persistScenario(t, db, minimalScenario(t, db, "second", "hash-second", "", "draft", 7, time.Minute))
	service := NewRolloverScenarioService(repository.NewRolloverScenarioRepository(db), nil, nil, nil, repository.NewAuditRepository(db), repository.NewTransactionManager(db))
	_, reused, err := service.Simulate(context.Background(), second.ID, first.IdempotencyKey, util.Actor{UserID: 7, Username: "operator", Role: string(constants.RolePKIOperator)}, "request-cross-key")
	if err == nil || reused {
		t.Fatalf("cross-scenario key got reused=%v err=%v", reused, err)
	}
	var apiErr *util.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusConflict || apiErr.Code != util.CodeIdempotency {
		t.Fatalf("got %#v, want 409 %s", err, util.CodeIdempotency)
	}
}

func TestCompareRejectsDifferentAlgorithmVersionAndFrozenInput(t *testing.T) {
	db := newScenarioTestDB(t)
	now := time.Date(2032, 6, 1, 12, 0, 0, 0, time.UTC)
	baseSnapshot := comparisonSnapshot(now, now.Add(time.Hour), now.Add(2*time.Hour))
	baseJSON, _ := baseSnapshot.Canonical()
	baseHash, _ := baseSnapshot.Hash()
	baseResult, err := algorithm.Simulate(baseSnapshot)
	if err != nil {
		t.Fatal(err)
	}

	differentVersion := minimalScenario(t, db, "older-algorithm", "hash-version-a", "version-key", "simulated", 7, 0)
	differentVersion.AlgorithmVersion = "trust-path-window-v0.9.0"
	differentVersion.InputSnapshot = baseJSON
	differentVersion.InputHash = "different-version-input"
	differentVersion.AffectedServicesJSON, differentVersion.BrokenPathsJSON, differentVersion.PathEvidenceJSON = mustResultJSON(baseResult)
	differentVersion = persistScenario(t, db, differentVersion)

	baseline := minimalScenario(t, db, "baseline-window", baseHash, "baseline-key", "simulated", 7, time.Minute)
	baseline.AlgorithmVersion = algorithm.Version
	baseline.InputSnapshot = baseJSON
	baseline.InputHash = baseHash
	baseline.AffectedServicesJSON, baseline.BrokenPathsJSON, baseline.PathEvidenceJSON = mustResultJSON(baseResult)
	baseline = persistScenario(t, db, baseline)

	shiftedSnapshot := comparisonSnapshot(now, now.Add(3*time.Hour), now.Add(4*time.Hour))
	shiftedJSON, _ := shiftedSnapshot.Canonical()
	shiftedHash, _ := shiftedSnapshot.Hash()
	if shiftedHash == baseHash {
		t.Fatal("window change should alter the full input hash")
	}
	frozenBase, _ := baseSnapshot.FrozenInputHash()
	frozenShifted, _ := shiftedSnapshot.FrozenInputHash()
	if frozenBase != frozenShifted {
		t.Fatal("frozen topology hash must match for window-only changes")
	}
	shiftedResult, err := algorithm.Simulate(shiftedSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	shifted := minimalScenario(t, db, "shifted-window", shiftedHash, "shifted-key", "ready", 7, 2*time.Minute)
	shifted.AlgorithmVersion = algorithm.Version
	shifted.InputSnapshot = shiftedJSON
	shifted.InputHash = shiftedHash
	shifted.AffectedServicesJSON, shifted.BrokenPathsJSON, shifted.PathEvidenceJSON = mustResultJSON(shiftedResult)
	shifted = persistScenario(t, db, shifted)

	changedTopology := comparisonSnapshot(now, now.Add(time.Hour), now.Add(2*time.Hour))
	changedTopology.Services[0].TrustAnchorIDs = []uint{2}
	changedJSON, _ := changedTopology.Canonical()
	changedResult, err := algorithm.Simulate(changedTopology)
	if err != nil {
		t.Fatal(err)
	}
	changed := minimalScenario(t, db, "changed-topology", "changed-topology-hash", "changed-key", "simulated", 7, 3*time.Minute)
	changed.AlgorithmVersion = algorithm.Version
	changed.InputSnapshot = changedJSON
	changed.InputHash = "changed-topology-hash"
	changed.AffectedServicesJSON, changed.BrokenPathsJSON, changed.PathEvidenceJSON = mustResultJSON(changedResult)
	changed = persistScenario(t, db, changed)

	draft := minimalScenario(t, db, "still-draft", "draft-hash", "", "draft", 7, 4*time.Minute)
	draft.AlgorithmVersion = algorithm.Version
	draft.InputSnapshot = baseJSON
	draft = persistScenario(t, db, draft)

	service := NewRolloverScenarioService(repository.NewRolloverScenarioRepository(db), nil, nil, nil, repository.NewAuditRepository(db), repository.NewTransactionManager(db))

	assertConflict := func(otherID uint, label string) {
		t.Helper()
		_, err := service.Compare(context.Background(), baseline.ID, otherID)
		var apiErr *util.APIError
		if !errors.As(err, &apiErr) || apiErr.Status != http.StatusConflict || apiErr.Code != util.CodeScenarioNotComparable {
			t.Fatalf("%s: got %#v, want 409 %s", label, err, util.CodeScenarioNotComparable)
		}
	}
	assertConflict(differentVersion.ID, "algorithm mismatch")
	assertConflict(changed.ID, "frozen input mismatch")
	assertConflict(draft.ID, "draft scenario")

	if _, err := service.Compare(context.Background(), baseline.ID, baseline.ID); err == nil {
		t.Fatal("comparing a scenario with itself should be rejected")
	}

	comparison, err := service.Compare(context.Background(), baseline.ID, shifted.ID)
	if err != nil {
		t.Fatalf("window-only comparison should succeed: %v", err)
	}
	if comparison.First.ScenarioID != baseline.ID || comparison.Second.ScenarioID != shifted.ID {
		t.Fatalf("comparison sides misassigned: %+v", comparison)
	}
	if comparison.First.CriticalAffectedCount < 0 || comparison.Second.CriticalAffectedCount < 0 {
		t.Fatal("critical damaged counts must be present on both sides")
	}
}

func comparisonSnapshot(now, start, end time.Time) algorithm.Snapshot {
	return algorithm.NewSnapshot(
		algorithm.ScenarioConfig{Name: "compare", OldAnchorID: 1, NewAnchorID: 2, OverlapStart: start, OverlapEnd: end, CandidateChainIDs: []uint{1, 2}, SimulationTime: now.Add(30 * 24 * time.Hour)},
		[]algorithm.AnchorSnapshot{
			{ID: 1, Code: "OLD", State: "valid", NotBefore: now.AddDate(-3, 0, 0), NotAfter: now.AddDate(0, 6, 0)},
			{ID: 2, Code: "NEW", State: "valid", NotBefore: now.AddDate(0, 0, -1), NotAfter: now.AddDate(6, 0, 0)},
		},
		[]algorithm.ChainSnapshot{
			{ID: 1, Code: "OLD-CHAIN", AnchorID: 1, LeafSubject: "CN=api", ValidFrom: now.AddDate(-1, 0, 0), ValidTo: now.AddDate(0, 4, 0), State: "validated", ValidationValid: true},
			{ID: 2, Code: "NEW-CHAIN", AnchorID: 2, LeafSubject: "CN=api", ValidFrom: now.AddDate(0, 0, -1), ValidTo: now.AddDate(2, 0, 0), State: "validated", ValidationValid: true},
		},
		[]algorithm.ServiceSnapshot{{ID: 1, Code: "EDGE", ChainID: 1, TrustAnchorIDs: []uint{1}, Criticality: "critical", State: "active"}},
	)
}

func mustResultJSON(result algorithm.Result) (string, string, string) {
	affected, _ := encode(result.AffectedServices)
	paths, _ := encode(result.BrokenPaths)
	evidence, _ := encode(result.Evidence)
	return affected, paths, evidence
}

func TestReplayEvidenceRollsBackWhenAuditAppendFails(t *testing.T) {
	db := newScenarioTestDB(t)
	now := time.Date(2032, 6, 1, 12, 0, 0, 0, time.UTC)
	snapshot := algorithm.NewSnapshot(
		algorithm.ScenarioConfig{Name: "replay", OldAnchorID: 1, NewAnchorID: 2, OverlapStart: now.Add(time.Hour), OverlapEnd: now.Add(2 * time.Hour), CandidateChainIDs: []uint{1}, SimulationTime: now.Add(90 * time.Minute)},
		[]algorithm.AnchorSnapshot{{ID: 1, Code: "OLD", State: "valid", NotBefore: now.Add(-time.Hour), NotAfter: now.Add(24 * time.Hour)}, {ID: 2, Code: "NEW", State: "valid", NotBefore: now.Add(-time.Hour), NotAfter: now.Add(24 * time.Hour)}},
		[]algorithm.ChainSnapshot{{ID: 1, Code: "CHAIN", AnchorID: 1, LeafSubject: "CN=leaf", ValidFrom: now.Add(-time.Hour), ValidTo: now.Add(24 * time.Hour), State: "validated", ValidationValid: true}},
		[]algorithm.ServiceSnapshot{{ID: 1, Code: "SERVICE", ChainID: 1, TrustAnchorIDs: []uint{1}, Criticality: "critical", State: "active"}},
	)
	result, err := algorithm.Simulate(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	snapshotJSON, _ := snapshot.Canonical()
	hash, _ := snapshot.Hash()
	affectedJSON, _ := encode(result.AffectedServices)
	pathsJSON, _ := encode(result.BrokenPaths)
	evidenceJSON, _ := encode(result.Evidence)
	base := minimalScenario(t, db, "replay", hash, "replay-key", "simulated", 7, 5*time.Minute)
	base.InputSnapshot, base.AffectedServicesJSON, base.BrokenPathsJSON, base.PathEvidenceJSON, base.Explanation = snapshotJSON, affectedJSON, pathsJSON, evidenceJSON, result.Explanation
	scenario := persistScenario(t, db, base)
	auditErr := errors.New("audit storage unavailable")
	service := NewRolloverScenarioService(repository.NewRolloverScenarioRepository(db), nil, nil, nil, failingAuditRepository{err: auditErr}, repository.NewTransactionManager(db))
	_, err = service.Replay(context.Background(), scenario.ID, util.Actor{UserID: 7, Username: "operator", Role: string(constants.RolePKIOperator)}, "request-replay")
	if err == nil {
		t.Fatal("replay should fail when audit append fails")
	}
	var stored model.RolloverScenario
	if err := db.First(&stored, scenario.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.ReplayVerified {
		t.Fatal("replay_verified committed without its audit record")
	}
}
