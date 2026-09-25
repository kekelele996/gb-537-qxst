package dto

import (
	"encoding/json"
	"time"

	"pki-certificate-rollover-impact/backend/internal/algorithm"
	"pki-certificate-rollover-impact/backend/internal/model"
)

type CreateRolloverScenarioRequest struct {
	Name              string    `json:"name" validate:"required,min=3,max=180"`
	OldAnchorID       uint      `json:"old_anchor_id" validate:"required"`
	NewAnchorID       uint      `json:"new_anchor_id" validate:"required"`
	OverlapStart      time.Time `json:"overlap_start" validate:"required"`
	OverlapEnd        time.Time `json:"overlap_end" validate:"required"`
	CandidateChainIDs []uint    `json:"candidate_chain_ids" validate:"required,min=1,max=32,dive,gt=0"`
	SimulationTime    time.Time `json:"simulation_time" validate:"required"`
}
type RolloverScenarioTransitionRequest struct {
	ToState string `json:"to_state" validate:"required,oneof=draft ready executing verified rollback"`
	Comment string `json:"comment" validate:"max=1000"`
}
type RolloverScenarioQuery struct {
	State     string
	CreatedBy uint
	Page      int
	PageSize  int
}

type RolloverScenarioResponse struct {
	ID                   uint                          `json:"id"`
	Name                 string                        `json:"name"`
	OldAnchorID          uint                          `json:"old_anchor_id"`
	NewAnchorID          uint                          `json:"new_anchor_id"`
	OldAnchor            *TrustAnchorResponse          `json:"old_anchor,omitempty"`
	NewAnchor            *TrustAnchorResponse          `json:"new_anchor,omitempty"`
	OverlapStart         time.Time                     `json:"overlap_start"`
	OverlapEnd           time.Time                     `json:"overlap_end"`
	CandidateChainIDs    []uint                        `json:"candidate_chain_ids"`
	AlgorithmVersion     string                        `json:"algorithm_version"`
	InputHash            string                        `json:"input_hash"`
	SimulationTime       time.Time                     `json:"simulation_time"`
	AffectedServicesJSON []algorithm.AffectedService   `json:"affected_services_json"`
	BrokenPathsJSON      []algorithm.BrokenPath        `json:"broken_paths_json"`
	PathEvidenceJSON     []algorithm.TimepointEvidence `json:"path_evidence_json"`
	ScenarioState        string                        `json:"scenario_state"`
	Explanation          string                        `json:"explanation"`
	CreatedBy            uint                          `json:"created_by"`
	CreatedByName        string                        `json:"created_by_name"`
	VerifiedBy           *uint                         `json:"verified_by,omitempty"`
	VerifiedByName       string                        `json:"verified_by_name"`
	ReplayVerified       bool                          `json:"replay_verified"`
	DurationMS           int64                         `json:"duration_ms"`
	RollbackRecord       string                        `json:"rollback_record"`
	CreatedAt            time.Time                     `json:"created_at"`
	UpdatedAt            time.Time                     `json:"updated_at"`
}
type RolloverScenarioListResponse struct {
	Items []RolloverScenarioResponse `json:"items"`
	Total int64                      `json:"total"`
	Page  int                        `json:"page"`
	Size  int                        `json:"size"`
}

func NewRolloverScenarioResponse(scenario model.RolloverScenario, now time.Time) RolloverScenarioResponse {
	candidateIDs := []uint{}
	affected := []algorithm.AffectedService{}
	paths := []algorithm.BrokenPath{}
	evidence := []algorithm.TimepointEvidence{}
	_ = json.Unmarshal([]byte(scenario.CandidateChainIDs), &candidateIDs)
	_ = json.Unmarshal([]byte(scenario.AffectedServicesJSON), &affected)
	_ = json.Unmarshal([]byte(scenario.BrokenPathsJSON), &paths)
	_ = json.Unmarshal([]byte(scenario.PathEvidenceJSON), &evidence)
	response := RolloverScenarioResponse{ID: scenario.ID, Name: scenario.Name, OldAnchorID: scenario.OldAnchorID, NewAnchorID: scenario.NewAnchorID, OverlapStart: scenario.OverlapStart, OverlapEnd: scenario.OverlapEnd, CandidateChainIDs: candidateIDs, AlgorithmVersion: scenario.AlgorithmVersion, InputHash: scenario.InputHash, SimulationTime: scenario.SimulationTime, AffectedServicesJSON: affected, BrokenPathsJSON: paths, PathEvidenceJSON: evidence, ScenarioState: scenario.ScenarioState, Explanation: scenario.Explanation, CreatedBy: scenario.CreatedBy, CreatedByName: scenario.CreatedByName, VerifiedBy: scenario.VerifiedBy, VerifiedByName: scenario.VerifiedByName, ReplayVerified: scenario.ReplayVerified, DurationMS: scenario.DurationMS, RollbackRecord: scenario.RollbackRecord, CreatedAt: scenario.CreatedAt, UpdatedAt: scenario.UpdatedAt}
	if scenario.OldAnchor.ID != 0 {
		anchor := NewTrustAnchorResponse(scenario.OldAnchor, 0, now)
		response.OldAnchor = &anchor
	}
	if scenario.NewAnchor.ID != 0 {
		anchor := NewTrustAnchorResponse(scenario.NewAnchor, 0, now)
		response.NewAnchor = &anchor
	}
	return response
}
