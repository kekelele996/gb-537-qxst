package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-playground/validator/v10"
	"net/http"
	"pki-certificate-rollover-impact/backend/internal/algorithm"
	"pki-certificate-rollover-impact/backend/internal/model"
	"pki-certificate-rollover-impact/backend/internal/repository"
	"pki-certificate-rollover-impact/backend/internal/util"
	"sort"
	"time"
)

var validate = validator.New()

func validateRequest(value any) error {
	if err := validate.Struct(value); err != nil {
		return util.WrapError(http.StatusBadRequest, util.CodeValidation, "request validation failed", err)
	}
	return nil
}
func encode(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode JSON: %w", err)
	}
	return string(raw), nil
}
func decodeUintList(raw string) ([]uint, error) {
	values := []uint{}
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("decode identifier list: %w", err)
	}
	return uniqueIDs(values), nil
}
func uniqueIDs(values []uint) []uint {
	seen := map[uint]bool{}
	result := make([]uint, 0, len(values))
	for _, value := range values {
		if value > 0 && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
func recordAudit(ctx context.Context, audits repository.AuditRepository, actor util.Actor, requestID, entity string, id uint, action string, before, after any, inputHash, version string, simulationTime *time.Time, duration int64, pathSummary string) error {
	entry := model.AuditLog{RequestID: requestID, ActorID: actor.UserID, ActorName: actor.Username, ActorRole: actor.Role, EntityType: entity, EntityID: id, Action: action, BeforeSnapshot: util.RedactedJSON(before), AfterSnapshot: util.RedactedJSON(after), InputHash: inputHash, AlgorithmVersion: version, SimulationTime: simulationTime, DurationMS: duration, PathSummary: util.CompactText(pathSummary, 1200), CreatedAt: time.Now().UTC()}
	if err := audits.Record(ctx, entry); err != nil {
		return util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to record immutable audit evidence", err)
	}
	return nil
}

func servicesContainCycle(services []model.DependentService) error {
	edges := map[uint][]uint{}
	for _, service := range services {
		ids, err := decodeUintList(service.DependencyEdgesJSON)
		if err != nil {
			return err
		}
		edges[service.ID] = ids
	}
	state := map[uint]int{}
	var visit func(uint) error
	visit = func(id uint) error {
		if state[id] == 1 {
			return fmt.Errorf("dependency graph contains a cycle at service %d", id)
		}
		if state[id] == 2 {
			return nil
		}
		state[id] = 1
		for _, next := range edges[id] {
			if _, exists := edges[next]; !exists {
				return fmt.Errorf("dependency service %d does not exist", next)
			}
			if err := visit(next); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	for id := range edges {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

type AuditService struct{ audits repository.AuditRepository }

func NewAuditService(audits repository.AuditRepository) *AuditService {
	return &AuditService{audits: audits}
}

func (s *AuditService) List(ctx context.Context, query repository.AuditQuery) (map[string]any, error) {
	entries, total, err := s.audits.List(ctx, query)
	if err != nil {
		return nil, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to list audit logs", err)
	}
	return map[string]any{"items": entries, "total": total, "page": query.Page, "size": query.PageSize}, nil
}

func buildSnapshot(name string, oldAnchorID, newAnchorID uint, overlapStart, overlapEnd, simulationTime time.Time, candidateIDs []uint, anchors []model.TrustAnchor, chains []model.CertificateChain, services []model.DependentService) (algorithm.Snapshot, error) {
	anchorSnapshots := make([]algorithm.AnchorSnapshot, 0, len(anchors))
	for _, anchor := range anchors {
		anchorSnapshots = append(anchorSnapshots, algorithm.AnchorSnapshot{ID: anchor.ID, Code: anchor.AnchorCode, State: anchor.CertificateState, NotBefore: anchor.NotBefore, NotAfter: anchor.NotAfter, Revoked: anchor.RevokedAt != nil})
	}
	chainSnapshots := make([]algorithm.ChainSnapshot, 0, len(chains))
	for _, chain := range chains {
		var evidence struct {
			Valid bool `json:"valid"`
		}
		_ = json.Unmarshal([]byte(chain.ValidationResult), &evidence)
		chainSnapshots = append(chainSnapshots, algorithm.ChainSnapshot{ID: chain.ID, Code: chain.ChainCode, AnchorID: chain.TrustAnchorID, LeafSubject: chain.LeafSubject, ValidFrom: chain.ValidFrom, ValidTo: chain.ValidTo, State: chain.ChainState, ValidationValid: evidence.Valid})
	}
	serviceSnapshots := make([]algorithm.ServiceSnapshot, 0, len(services))
	for _, service := range services {
		trust, err := decodeUintList(service.ClientTrustRefsJSON)
		if err != nil {
			return algorithm.Snapshot{}, err
		}
		dependencies, err := decodeUintList(service.DependencyEdgesJSON)
		if err != nil {
			return algorithm.Snapshot{}, err
		}
		serviceSnapshots = append(serviceSnapshots, algorithm.ServiceSnapshot{ID: service.ID, Code: service.ServiceCode, ChainID: service.ChainID, TrustAnchorIDs: trust, DependencyIDs: dependencies, Criticality: service.Criticality, State: service.ServiceState})
	}
	snapshot := algorithm.NewSnapshot(algorithm.ScenarioConfig{Name: name, OldAnchorID: oldAnchorID, NewAnchorID: newAnchorID, OverlapStart: overlapStart.UTC(), OverlapEnd: overlapEnd.UTC(), CandidateChainIDs: candidateIDs, SimulationTime: simulationTime.UTC()}, anchorSnapshots, chainSnapshots, serviceSnapshots)
	if err := algorithm.ValidateSnapshot(snapshot); err != nil {
		return algorithm.Snapshot{}, err
	}
	return snapshot, nil
}
