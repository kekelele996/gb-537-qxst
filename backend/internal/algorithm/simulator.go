package algorithm

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"pki-certificate-rollover-impact/backend/internal/util"
)

const Version = "trust-path-window-v1.0.0"

type AnchorSnapshot struct {
	ID        uint      `json:"id"`
	Code      string    `json:"code"`
	State     string    `json:"state"`
	NotBefore time.Time `json:"not_before"`
	NotAfter  time.Time `json:"not_after"`
	Revoked   bool      `json:"revoked"`
}
type ChainSnapshot struct {
	ID              uint      `json:"id"`
	Code            string    `json:"code"`
	AnchorID        uint      `json:"anchor_id"`
	LeafSubject     string    `json:"leaf_subject"`
	ValidFrom       time.Time `json:"valid_from"`
	ValidTo         time.Time `json:"valid_to"`
	State           string    `json:"state"`
	ValidationValid bool      `json:"validation_valid"`
}
type ServiceSnapshot struct {
	ID             uint   `json:"id"`
	Code           string `json:"code"`
	ChainID        uint   `json:"chain_id"`
	TrustAnchorIDs []uint `json:"trust_anchor_ids"`
	DependencyIDs  []uint `json:"dependency_ids"`
	Criticality    string `json:"criticality"`
	State          string `json:"state"`
}
type ScenarioConfig struct {
	Name              string    `json:"name"`
	OldAnchorID       uint      `json:"old_anchor_id"`
	NewAnchorID       uint      `json:"new_anchor_id"`
	OverlapStart      time.Time `json:"overlap_start"`
	OverlapEnd        time.Time `json:"overlap_end"`
	CandidateChainIDs []uint    `json:"candidate_chain_ids"`
	SimulationTime    time.Time `json:"simulation_time"`
}
type Snapshot struct {
	AlgorithmVersion string            `json:"algorithm_version"`
	Config           ScenarioConfig    `json:"config"`
	Anchors          []AnchorSnapshot  `json:"anchors"`
	Chains           []ChainSnapshot   `json:"chains"`
	Services         []ServiceSnapshot `json:"services"`
}

type AffectedService struct {
	ServiceID   uint      `json:"service_id"`
	ServiceCode string    `json:"service_code"`
	Criticality string    `json:"criticality"`
	At          time.Time `json:"at"`
	Reason      string    `json:"reason"`
}
type BrokenPath struct {
	At           time.Time `json:"at"`
	ServiceCodes []string  `json:"service_codes"`
	Reason       string    `json:"reason"`
}
type ServiceEvidence struct {
	ServiceID        uint   `json:"service_id"`
	ServiceCode      string `json:"service_code"`
	Reachable        bool   `json:"reachable"`
	SelectedChainID  uint   `json:"selected_chain_id,omitempty"`
	SelectedAnchorID uint   `json:"selected_anchor_id,omitempty"`
	Reason           string `json:"reason"`
}
type TimepointEvidence struct {
	At              time.Time         `json:"at"`
	ActiveAnchorIDs []uint            `json:"active_anchor_ids"`
	Services        []ServiceEvidence `json:"services"`
}
type Result struct {
	AffectedServices []AffectedService   `json:"affected_services"`
	BrokenPaths      []BrokenPath        `json:"broken_paths"`
	Evidence         []TimepointEvidence `json:"evidence"`
	Explanation      string              `json:"explanation"`
}

func NewSnapshot(config ScenarioConfig, anchors []AnchorSnapshot, chains []ChainSnapshot, services []ServiceSnapshot) Snapshot {
	config.CandidateChainIDs = sortedUnique(config.CandidateChainIDs)
	sort.Slice(anchors, func(i, j int) bool { return anchors[i].ID < anchors[j].ID })
	sort.Slice(chains, func(i, j int) bool { return chains[i].ID < chains[j].ID })
	sort.Slice(services, func(i, j int) bool { return services[i].ID < services[j].ID })
	for index := range services {
		services[index].TrustAnchorIDs = sortedUnique(services[index].TrustAnchorIDs)
		services[index].DependencyIDs = sortedUnique(services[index].DependencyIDs)
	}
	return Snapshot{AlgorithmVersion: Version, Config: config, Anchors: anchors, Chains: chains, Services: services}
}

func (snapshot Snapshot) Canonical() (string, error) { return util.CanonicalJSON(snapshot) }
func (snapshot Snapshot) Hash() (string, error) {
	raw, err := snapshot.Canonical()
	if err != nil {
		return "", err
	}
	return util.HashString(raw), nil
}
func DecodeSnapshot(raw string) (Snapshot, error) {
	var snapshot Snapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("decode rollover snapshot: %w", err)
	}
	return snapshot, nil
}

func ValidateSnapshot(snapshot Snapshot) error {
	cfg := snapshot.Config
	if cfg.OldAnchorID == 0 || cfg.NewAnchorID == 0 || cfg.OldAnchorID == cfg.NewAnchorID {
		return fmt.Errorf("old and new trust anchors must be distinct")
	}
	if !cfg.OverlapStart.Before(cfg.OverlapEnd) {
		return fmt.Errorf("overlap_start must be before overlap_end")
	}
	if cfg.SimulationTime.IsZero() {
		return fmt.Errorf("simulation_time is required")
	}
	anchorSeen := map[uint]bool{}
	for _, anchor := range snapshot.Anchors {
		anchorSeen[anchor.ID] = true
	}
	if !anchorSeen[cfg.OldAnchorID] || !anchorSeen[cfg.NewAnchorID] {
		return fmt.Errorf("scenario anchors are missing from frozen input")
	}
	chainSeen := map[uint]bool{}
	for _, chain := range snapshot.Chains {
		chainSeen[chain.ID] = true
	}
	for _, id := range cfg.CandidateChainIDs {
		if !chainSeen[id] {
			return fmt.Errorf("candidate chain %d is missing", id)
		}
	}
	if err := detectCycle(snapshot.Services); err != nil {
		return err
	}
	return nil
}

func Simulate(snapshot Snapshot) (Result, error) {
	if err := ValidateSnapshot(snapshot); err != nil {
		return Result{}, err
	}
	points := criticalTimes(snapshot.Config)
	chainByID := map[uint]ChainSnapshot{}
	serviceByID := map[uint]ServiceSnapshot{}
	for _, chain := range snapshot.Chains {
		chainByID[chain.ID] = chain
	}
	for _, service := range snapshot.Services {
		serviceByID[service.ID] = service
	}
	result := Result{AffectedServices: []AffectedService{}, BrokenPaths: []BrokenPath{}, Evidence: []TimepointEvidence{}}
	for _, at := range points {
		active := activeAnchors(snapshot.Config, at)
		direct := map[uint]ServiceEvidence{}
		evidence := TimepointEvidence{At: at, ActiveAnchorIDs: mapKeys(active), Services: []ServiceEvidence{}}
		for _, service := range snapshot.Services {
			if service.State != "active" {
				continue
			}
			item := evaluateDirect(service, at, active, snapshot.Config.CandidateChainIDs, chainByID)
			direct[service.ID] = item
		}
		for _, service := range snapshot.Services {
			if service.State != "active" {
				continue
			}
			item, path := resolveService(service.ID, direct, serviceByID, map[uint]bool{}, nil)
			evidence.Services = append(evidence.Services, item)
			if !item.Reachable {
				result.AffectedServices = append(result.AffectedServices, AffectedService{ServiceID: service.ID, ServiceCode: service.Code, Criticality: service.Criticality, At: at, Reason: item.Reason})
				codes := make([]string, 0, len(path))
				for _, id := range path {
					if entry, ok := serviceByID[id]; ok {
						codes = append(codes, entry.Code)
					}
				}
				result.BrokenPaths = append(result.BrokenPaths, BrokenPath{At: at, ServiceCodes: codes, Reason: item.Reason})
			}
		}
		sort.Slice(evidence.Services, func(i, j int) bool { return evidence.Services[i].ServiceID < evidence.Services[j].ServiceID })
		result.Evidence = append(result.Evidence, evidence)
	}
	result.Explanation = explain(snapshot.Config, result)
	return result, nil
}

func evaluateDirect(service ServiceSnapshot, at time.Time, active map[uint]bool, candidates []uint, chains map[uint]ChainSnapshot) ServiceEvidence {
	current, ok := chains[service.ChainID]
	if !ok {
		return ServiceEvidence{ServiceID: service.ID, ServiceCode: service.Code, Reason: "configured certificate chain is missing"}
	}
	ids := append([]uint{service.ChainID}, candidates...)
	ids = sortedUnique(ids)
	reasons := []string{}
	for _, id := range ids {
		chain, exists := chains[id]
		if !exists || chain.LeafSubject != current.LeafSubject {
			continue
		}
		if chain.State == "revoked" || chain.State == "deprecated" {
			reasons = append(reasons, chain.Code+" is not active")
			continue
		}
		if at.Before(chain.ValidFrom) || !at.Before(chain.ValidTo) {
			reasons = append(reasons, chain.Code+" is outside its validity window")
			continue
		}
		if !chain.ValidationValid {
			reasons = append(reasons, chain.Code+" did not pass offline signature validation")
			continue
		}
		if !active[chain.AnchorID] {
			reasons = append(reasons, chain.Code+" anchor is unavailable at this timepoint")
			continue
		}
		if !contains(service.TrustAnchorIDs, chain.AnchorID) {
			reasons = append(reasons, service.Code+" trust set does not include anchor "+fmt.Sprint(chain.AnchorID))
			continue
		}
		return ServiceEvidence{ServiceID: service.ID, ServiceCode: service.Code, Reachable: true, SelectedChainID: id, SelectedAnchorID: chain.AnchorID, Reason: "certificate path is reachable"}
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "no candidate chain matches the configured leaf identity")
	}
	return ServiceEvidence{ServiceID: service.ID, ServiceCode: service.Code, Reason: strings.Join(reasons, "; ")}
}

func resolveService(id uint, direct map[uint]ServiceEvidence, services map[uint]ServiceSnapshot, visiting map[uint]bool, path []uint) (ServiceEvidence, []uint) {
	item, ok := direct[id]
	if !ok {
		return ServiceEvidence{ServiceID: id, Reason: "dependency service is absent or inactive"}, append(path, id)
	}
	currentPath := append(append([]uint{}, path...), id)
	if visiting[id] {
		item.Reachable = false
		item.Reason = "dependency cycle encountered in frozen graph"
		return item, currentPath
	}
	if !item.Reachable {
		return item, currentPath
	}
	visiting[id] = true
	defer delete(visiting, id)
	service := services[id]
	for _, dependencyID := range service.DependencyIDs {
		dep, brokenPath := resolveService(dependencyID, direct, services, visiting, currentPath)
		if !dep.Reachable {
			item.Reachable = false
			item.Reason = "upstream dependency " + dep.ServiceCode + " is unreachable: " + dep.Reason
			return item, brokenPath
		}
	}
	return item, currentPath
}

func criticalTimes(config ScenarioConfig) []time.Time {
	points := []time.Time{config.OverlapStart.Add(-time.Minute), config.OverlapStart, config.OverlapStart.Add(config.OverlapEnd.Sub(config.OverlapStart) / 2), config.OverlapEnd, config.OverlapEnd.Add(time.Minute), config.SimulationTime}
	sort.Slice(points, func(i, j int) bool { return points[i].Before(points[j]) })
	result := points[:0]
	for _, point := range points {
		if len(result) == 0 || !point.Equal(result[len(result)-1]) {
			result = append(result, point.UTC())
		}
	}
	return result
}
func activeAnchors(config ScenarioConfig, at time.Time) map[uint]bool {
	active := map[uint]bool{}
	if at.Before(config.OverlapStart) {
		active[config.OldAnchorID] = true
	} else if at.After(config.OverlapEnd) {
		active[config.NewAnchorID] = true
	} else {
		active[config.OldAnchorID] = true
		active[config.NewAnchorID] = true
	}
	return active
}
func explain(config ScenarioConfig, result Result) string {
	if len(result.AffectedServices) == 0 {
		return "All active service trust paths remain reachable before, during, and after the proposed overlap window."
	}
	before, after := 0, 0
	for _, item := range result.AffectedServices {
		if item.At.Before(config.OverlapStart) {
			before++
		}
		if item.At.After(config.OverlapEnd) {
			after++
		}
	}
	parts := []string{fmt.Sprintf("%d service-timepoint failures were found across %d broken paths.", len(result.AffectedServices), len(result.BrokenPaths))}
	if before > 0 {
		parts = append(parts, "Some paths fail before overlap, indicating the new trust path may be introduced too late.")
	}
	if after > 0 {
		parts = append(parts, "Some paths fail after overlap, indicating the old anchor may be removed before all clients trust the replacement.")
	}
	return strings.Join(parts, " ")
}
func Compare(first, second Result) map[string]any {
	return map[string]any{"first_affected": len(first.AffectedServices), "second_affected": len(second.AffectedServices), "affected_delta": len(second.AffectedServices) - len(first.AffectedServices), "first_broken_paths": len(first.BrokenPaths), "second_broken_paths": len(second.BrokenPaths)}
}
func detectCycle(services []ServiceSnapshot) error {
	edges := map[uint][]uint{}
	for _, service := range services {
		edges[service.ID] = service.DependencyIDs
	}
	state := map[uint]int{}
	var visit func(uint) error
	visit = func(id uint) error {
		if state[id] == 1 {
			return fmt.Errorf("service dependency graph contains a cycle at %d", id)
		}
		if state[id] == 2 {
			return nil
		}
		state[id] = 1
		for _, next := range edges[id] {
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
func sortedUnique(values []uint) []uint {
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
func contains(values []uint, target uint) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
func mapKeys(values map[uint]bool) []uint {
	result := make([]uint, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
