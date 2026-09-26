package service

import (
	"context"
	"errors"
	"net/http"
	"pki-certificate-rollover-impact/backend/internal/algorithm"
	"pki-certificate-rollover-impact/backend/internal/model"
	"pki-certificate-rollover-impact/backend/internal/repository"
	"pki-certificate-rollover-impact/backend/internal/util"
	"testing"
	"time"

	"gorm.io/gorm"
)

func compareTestSnapshot(name string, overlapEnd time.Time, dependencyIDs []uint) algorithm.Snapshot {
	start := time.Date(2032, 5, 10, 0, 0, 0, 0, time.UTC)
	return algorithm.NewSnapshot(
		algorithm.ScenarioConfig{Name: name, OldAnchorID: 1, NewAnchorID: 2, OverlapStart: start.Add(24 * time.Hour), OverlapEnd: overlapEnd, CandidateChainIDs: []uint{1, 2}, SimulationTime: start.Add(36 * time.Hour)},
		[]algorithm.AnchorSnapshot{{ID: 1, Code: "OLD", State: "valid", NotBefore: start.Add(-time.Hour), NotAfter: start.AddDate(1, 0, 0)}, {ID: 2, Code: "NEW", State: "valid", NotBefore: start.Add(-time.Hour), NotAfter: start.AddDate(1, 0, 0)}},
		[]algorithm.ChainSnapshot{{ID: 1, Code: "OLD-CHAIN", AnchorID: 1, LeafSubject: "CN=api", ValidFrom: start.Add(-time.Hour), ValidTo: start.AddDate(1, 0, 0), State: "validated", ValidationValid: true}, {ID: 2, Code: "NEW-CHAIN", AnchorID: 2, LeafSubject: "CN=api", ValidFrom: start.Add(-time.Hour), ValidTo: start.AddDate(1, 0, 0), State: "validated", ValidationValid: true}},
		[]algorithm.ServiceSnapshot{{ID: 1, Code: "gateway", ChainID: 1, TrustAnchorIDs: []uint{1, 2}, Criticality: "critical", State: "active"}, {ID: 2, Code: "client", ChainID: 1, TrustAnchorIDs: []uint{1}, DependencyIDs: dependencyIDs, Criticality: "high", State: "active"}},
	)
}

func persistSimulatedScenario(t *testing.T, db *gorm.DB, snapshot algorithm.Snapshot, offset time.Duration) model.RolloverScenario {
	t.Helper()
	result, err := algorithm.Simulate(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	snapshotJSON, _ := snapshot.Canonical()
	hash, _ := snapshot.Hash()
	affectedJSON, _ := encode(result.AffectedServices)
	pathsJSON, _ := encode(result.BrokenPaths)
	evidenceJSON, _ := encode(result.Evidence)
	base := minimalScenario(t, db, snapshot.Config.Name, hash, "compare-"+snapshot.Config.Name, "simulated", 7, offset)
	base.InputSnapshot = snapshotJSON
	base.AffectedServicesJSON = affectedJSON
	base.BrokenPathsJSON = pathsJSON
	base.PathEvidenceJSON = evidenceJSON
	base.Explanation = result.Explanation
	return persistScenario(t, db, base)
}

func compareService(db *gorm.DB) *RolloverScenarioService {
	return NewRolloverScenarioService(repository.NewRolloverScenarioRepository(db), nil, nil, nil, repository.NewAuditRepository(db), repository.NewTransactionManager(db))
}

func TestCompareRefusesDifferentAlgorithmVersions(t *testing.T) {
	db := newScenarioTestDB(t)
	overlapEnd := time.Date(2032, 5, 12, 0, 0, 0, 0, time.UTC)
	first := persistSimulatedScenario(t, db, compareTestSnapshot("plan-a", overlapEnd, nil), 0)
	second := persistSimulatedScenario(t, db, compareTestSnapshot("plan-b", overlapEnd.Add(6*time.Hour), nil), time.Minute)
	if err := db.Model(&second).Update("algorithm_version", "trust-path-window-v0.9.0").Error; err != nil {
		t.Fatal(err)
	}

	_, err := compareService(db).Compare(context.Background(), first.ID, second.ID)
	var apiErr *util.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusConflict || apiErr.Code != util.CodeCompareMismatch {
		t.Fatalf("got %#v, want 409 %s", err, util.CodeCompareMismatch)
	}
}

func TestCompareRefusesDifferentFrozenInventory(t *testing.T) {
	db := newScenarioTestDB(t)
	overlapEnd := time.Date(2032, 5, 12, 0, 0, 0, 0, time.UTC)
	first := persistSimulatedScenario(t, db, compareTestSnapshot("plan-a", overlapEnd, nil), 0)
	second := persistSimulatedScenario(t, db, compareTestSnapshot("plan-b", overlapEnd, []uint{1}), time.Minute)

	_, err := compareService(db).Compare(context.Background(), first.ID, second.ID)
	var apiErr *util.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusConflict || apiErr.Code != util.CodeCompareMismatch {
		t.Fatalf("got %#v, want 409 %s", err, util.CodeCompareMismatch)
	}
}

func TestCompareRefusesDraftScenario(t *testing.T) {
	db := newScenarioTestDB(t)
	overlapEnd := time.Date(2032, 5, 12, 0, 0, 0, 0, time.UTC)
	first := persistSimulatedScenario(t, db, compareTestSnapshot("plan-a", overlapEnd, nil), 0)
	draftSnapshot := compareTestSnapshot("plan-draft", overlapEnd.Add(6*time.Hour), nil)
	draftJSON, _ := draftSnapshot.Canonical()
	draftHash, _ := draftSnapshot.Hash()
	draft := minimalScenario(t, db, "plan-draft", draftHash, "compare-plan-draft", "draft", 7, 2*time.Minute)
	draft.InputSnapshot = draftJSON
	draft = persistScenario(t, db, draft)

	_, err := compareService(db).Compare(context.Background(), first.ID, draft.ID)
	var apiErr *util.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusConflict || apiErr.Code != util.CodeStateTransition {
		t.Fatalf("got %#v, want 409 %s", err, util.CodeStateTransition)
	}
}

func TestCompareReturnsServiceAndTimepointDelta(t *testing.T) {
	db := newScenarioTestDB(t)
	overlapEnd := time.Date(2032, 5, 12, 0, 0, 0, 0, time.UTC)
	firstSnapshot := compareTestSnapshot("plan-a", overlapEnd, nil)
	secondSnapshot := compareTestSnapshot("plan-b", overlapEnd.Add(24*time.Hour), nil)
	first := persistSimulatedScenario(t, db, firstSnapshot, 0)
	second := persistSimulatedScenario(t, db, secondSnapshot, time.Minute)

	comparison, err := compareService(db).Compare(context.Background(), first.ID, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	firstResult, err := algorithm.Simulate(firstSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	secondResult, err := algorithm.Simulate(secondSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	expected := algorithm.Diff(firstResult, secondResult)
	if len(comparison.NewImpacts) != len(expected.NewImpacts) || len(comparison.RecoveredImpacts) != len(expected.RecoveredImpacts) {
		t.Fatalf("delta mismatch: got %d new/%d recovered, want %d/%d", len(comparison.NewImpacts), len(comparison.RecoveredImpacts), len(expected.NewImpacts), len(expected.RecoveredImpacts))
	}
	if len(comparison.NewImpacts) == 0 || len(comparison.RecoveredImpacts) == 0 {
		t.Fatalf("adjusted overlap windows should produce both new and recovered impacts: %+v", comparison)
	}
	if comparison.FirstCriticalAffected != expected.FirstCriticalAffected || comparison.SecondCriticalAffected != expected.SecondCriticalAffected {
		t.Fatalf("critical counts got %d/%d, want %d/%d", comparison.FirstCriticalAffected, comparison.SecondCriticalAffected, expected.FirstCriticalAffected, expected.SecondCriticalAffected)
	}
	if comparison.InventoryHash == "" || comparison.AlgorithmVersion != algorithm.Version {
		t.Fatalf("comparison must pin the shared inventory hash and algorithm version: %+v", comparison)
	}
	if comparison.FirstID != first.ID || comparison.SecondID != second.ID || comparison.FirstName != "plan-a" || comparison.SecondName != "plan-b" {
		t.Fatalf("comparison identities are wrong: %+v", comparison)
	}
	for _, impact := range comparison.NewImpacts {
		if impact.ServiceCode == "" || impact.At.IsZero() {
			t.Fatalf("new impacts must be listed by service and timepoint: %+v", impact)
		}
	}
}
