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

func TestInventoryHashIgnoresScenarioConfig(t *testing.T) {
	first := syntheticSnapshot()
	second := syntheticSnapshot()
	second.Config.Name = "Adjusted overlap"
	second.Config.OverlapEnd = second.Config.OverlapEnd.Add(6 * time.Hour)
	second.Config.SimulationTime = second.Config.SimulationTime.Add(time.Hour)
	firstHash, err := first.InventoryHash()
	if err != nil {
		t.Fatal(err)
	}
	secondHash, err := second.InventoryHash()
	if err != nil {
		t.Fatal(err)
	}
	if firstHash != secondHash {
		t.Fatal("inventory hash must be stable across overlap-window adjustments")
	}
	second.Services[0].DependencyIDs = []uint{102}
	changed, err := second.InventoryHash()
	if err != nil {
		t.Fatal(err)
	}
	if changed == firstHash {
		t.Fatal("inventory hash must change when the frozen dependency graph changes")
	}
	firstFull, _ := first.Hash()
	secondFull, _ := second.Hash()
	if firstFull == secondFull {
		t.Fatal("full input hash should still capture config and inventory changes")
	}
}

func TestDiffSeparatesNewImpactsFromRecovered(t *testing.T) {
	at := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	later := at.Add(6 * time.Hour)
	first := Result{
		AffectedServices: []AffectedService{
			{ServiceID: 7, ServiceCode: "gateway", Criticality: "critical", At: at, Reason: "old anchor removed"},
			{ServiceID: 8, ServiceCode: "worker", Criticality: "low", At: at, Reason: "upstream unreachable"},
			{ServiceID: 9, ServiceCode: "audit", Criticality: "critical", At: later, Reason: "chain expired"},
		},
		BrokenPaths: []BrokenPath{
			{At: at, ServiceCodes: []string{"gateway", "worker"}, Reason: "old anchor removed"},
			{At: later, ServiceCodes: []string{"audit"}, Reason: "chain expired"},
		},
	}
	second := Result{
		AffectedServices: []AffectedService{
			{ServiceID: 7, ServiceCode: "gateway", Criticality: "critical", At: at, Reason: "old anchor removed"},
			{ServiceID: 10, ServiceCode: "billing", Criticality: "critical", At: later, Reason: "trust set does not include new anchor"},
		},
		BrokenPaths: []BrokenPath{
			{At: at, ServiceCodes: []string{"gateway", "worker"}, Reason: "old anchor removed"},
			{At: later, ServiceCodes: []string{"billing"}, Reason: "trust set does not include new anchor"},
		},
	}
	delta := Diff(first, second)
	if len(delta.NewImpacts) != 1 || delta.NewImpacts[0].ServiceCode != "billing" || !delta.NewImpacts[0].At.Equal(later) {
		t.Fatalf("expected billing as the only new impact: %+v", delta.NewImpacts)
	}
	if len(delta.RecoveredImpacts) != 2 {
		t.Fatalf("expected worker and audit to recover: %+v", delta.RecoveredImpacts)
	}
	if len(delta.NewBrokenPaths) != 1 || delta.NewBrokenPaths[0].ServiceCodes[0] != "billing" {
		t.Fatalf("expected billing path as new: %+v", delta.NewBrokenPaths)
	}
	if len(delta.ResolvedBrokenPaths) != 1 || delta.ResolvedBrokenPaths[0].ServiceCodes[0] != "audit" {
		t.Fatalf("expected audit path resolved: %+v", delta.ResolvedBrokenPaths)
	}
	if delta.FirstCriticalAffected != 2 || delta.SecondCriticalAffected != 2 {
		t.Fatalf("critical counts got %d/%d, want 2/2", delta.FirstCriticalAffected, delta.SecondCriticalAffected)
	}
	again := Diff(first, second)
	if len(again.NewImpacts) != 1 || again.NewImpacts[0].ServiceCode != "billing" {
		t.Fatal("diff must be deterministic")
	}
}
