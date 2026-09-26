package algorithm

import (
	"strings"
	"testing"
	"time"
)

func syntheticSnapshot() Snapshot {
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	return NewSnapshot(ScenarioConfig{Name: "Synthetic rollover", OldAnchorID: 1, NewAnchorID: 2, OverlapStart: start, OverlapEnd: start.Add(24 * time.Hour), CandidateChainIDs: []uint{11, 10, 11}, SimulationTime: start.Add(12 * time.Hour)}, []AnchorSnapshot{{ID: 2, Code: "new", State: "valid", NotBefore: start.Add(-time.Hour), NotAfter: start.AddDate(2, 0, 0)}, {ID: 1, Code: "old", State: "valid", NotBefore: start.AddDate(-2, 0, 0), NotAfter: start.AddDate(0, 3, 0)}}, []ChainSnapshot{{ID: 11, Code: "new-chain", AnchorID: 2, LeafSubject: "CN=api", ValidFrom: start.Add(-time.Hour), ValidTo: start.AddDate(1, 0, 0), State: "validated", ValidationValid: true}, {ID: 10, Code: "old-chain", AnchorID: 1, LeafSubject: "CN=api", ValidFrom: start.AddDate(-1, 0, 0), ValidTo: start.AddDate(0, 2, 0), State: "validated", ValidationValid: true}}, []ServiceSnapshot{{ID: 101, Code: "gateway", ChainID: 10, TrustAnchorIDs: []uint{2, 1}, Criticality: "critical", State: "active"}, {ID: 102, Code: "client", ChainID: 10, TrustAnchorIDs: []uint{1}, DependencyIDs: []uint{101}, Criticality: "high", State: "active"}})
}

func TestSimulationIsDeterministicAndExplainsTrustGap(t *testing.T) {
	snapshot := syntheticSnapshot()
	firstHash, err := snapshot.Hash()
	if err != nil {
		t.Fatal(err)
	}
	secondHash, err := snapshot.Hash()
	if err != nil || firstHash != secondHash {
		t.Fatalf("hash is not deterministic: %s %s %v", firstHash, secondHash, err)
	}
	first, err := Simulate(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Simulate(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.AffectedServices) == 0 || len(first.BrokenPaths) == 0 {
		t.Fatalf("expected broken trust paths: %+v", first)
	}
	if first.Explanation != second.Explanation || len(first.Evidence) != len(second.Evidence) {
		t.Fatal("replay result differs")
	}
	found := false
	for _, item := range first.AffectedServices {
		if item.ServiceCode == "client" && strings.Contains(item.Reason, "trust set") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected client trust-set explanation: %+v", first.AffectedServices)
	}
}

func TestCycleIsRejected(t *testing.T) {
	snapshot := syntheticSnapshot()
	snapshot.Services[0].DependencyIDs = []uint{102}
	if _, err := Simulate(snapshot); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestFrozenInputHashIgnoresOverlapWindow(t *testing.T) {
	baseline := syntheticSnapshot()
	shifted := syntheticSnapshot()
	shifted.Config.OverlapStart = baseline.Config.OverlapStart.Add(48 * time.Hour)
	shifted.Config.OverlapEnd = baseline.Config.OverlapEnd.Add(48 * time.Hour)
	baselineHash, err := baseline.FrozenInputHash()
	if err != nil {
		t.Fatal(err)
	}
	shiftedHash, err := shifted.FrozenInputHash()
	if err != nil {
		t.Fatal(err)
	}
	if baselineHash != shiftedHash {
		t.Fatal("frozen input hash must stay stable when only the overlap window changes")
	}
	if baseline.AlgorithmVersion != shifted.AlgorithmVersion {
		t.Fatal("algorithm versions should match for identical snapshots")
	}
}

func TestDiffResultsDistinguishesNewResolvedAndCriticalDamage(t *testing.T) {
	firstAt := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	secondAt := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	first := Result{
		AffectedServices: []AffectedService{
			{ServiceID: 101, ServiceCode: "gateway", Criticality: "critical", At: firstAt, Reason: "old root removed"},
			{ServiceID: 102, ServiceCode: "worker", Criticality: "high", At: firstAt, Reason: "upstream unreachable"},
		},
		BrokenPaths: []BrokenPath{
			{At: firstAt, ServiceCodes: []string{"gateway"}, Reason: "old root removed"},
			{At: firstAt, ServiceCodes: []string{"worker", "gateway"}, Reason: "upstream unreachable"},
		},
	}
	second := Result{
		AffectedServices: []AffectedService{
			{ServiceID: 101, ServiceCode: "gateway", Criticality: "critical", At: firstAt, Reason: "old root removed"},
			{ServiceID: 103, ServiceCode: "billing", Criticality: "critical", At: secondAt, Reason: "new anchor unavailable"},
		},
		BrokenPaths: []BrokenPath{
			{At: firstAt, ServiceCodes: []string{"gateway"}, Reason: "old root removed"},
			{At: secondAt, ServiceCodes: []string{"billing"}, Reason: "new anchor unavailable"},
		},
	}
	comparison := DiffResults(
		SideSummary(1, "baseline", "simulated", firstAt, firstAt.Add(time.Hour), first),
		SideSummary(2, "adjusted", "ready", secondAt, secondAt.Add(time.Hour), second),
		first, second,
	)
	if len(comparison.NewImpacts) != 1 || comparison.NewImpacts[0].ServiceCode != "billing" {
		t.Fatalf("expected only billing as a new impact, got %+v", comparison.NewImpacts)
	}
	if len(comparison.ResolvedImpacts) != 1 || comparison.ResolvedImpacts[0].ServiceCode != "worker" {
		t.Fatalf("expected worker recovery, got %+v", comparison.ResolvedImpacts)
	}
	if len(comparison.IntroducedBrokenPaths) != 1 || comparison.IntroducedBrokenPaths[0].ServiceCodes[0] != "billing" {
		t.Fatalf("expected billing path introduced, got %+v", comparison.IntroducedBrokenPaths)
	}
	if len(comparison.EliminatedBrokenPaths) != 1 || len(comparison.EliminatedBrokenPaths[0].ServiceCodes) != 2 {
		t.Fatalf("expected worker->gateway path eliminated, got %+v", comparison.EliminatedBrokenPaths)
	}
	if comparison.First.CriticalAffectedCount != 1 || comparison.Second.CriticalAffectedCount != 2 {
		t.Fatalf("critical damaged counts wrong: %d vs %d", comparison.First.CriticalAffectedCount, comparison.Second.CriticalAffectedCount)
	}
	if comparison.First.CriticalityBreakdown.High != 1 || comparison.Second.CriticalityBreakdown.Critical != 2 {
		t.Fatalf("criticality breakdown wrong: %+v %+v", comparison.First.CriticalityBreakdown, comparison.Second.CriticalityBreakdown)
	}
}

func TestDiffResultsTreatsSameServiceAtDifferentTimepointAsNew(t *testing.T) {
	firstAt := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	secondAt := firstAt.Add(time.Hour)
	first := Result{AffectedServices: []AffectedService{{ServiceID: 101, ServiceCode: "gateway", Criticality: "critical", At: firstAt, Reason: "gap"}}, BrokenPaths: []BrokenPath{}}
	second := Result{AffectedServices: []AffectedService{{ServiceID: 101, ServiceCode: "gateway", Criticality: "critical", At: secondAt, Reason: "gap"}}, BrokenPaths: []BrokenPath{}}
	comparison := DiffResults(ComparisonSide{}, ComparisonSide{}, first, second)
	if len(comparison.NewImpacts) != 1 || len(comparison.ResolvedImpacts) != 1 {
		t.Fatalf("service-timepoint key must distinguish timepoints: %+v", comparison)
	}
}
