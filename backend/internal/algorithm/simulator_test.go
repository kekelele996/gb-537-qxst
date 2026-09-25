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
