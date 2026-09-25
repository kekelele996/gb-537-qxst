package model

import "time"

type RolloverScenario struct {
	ID                   uint        `gorm:"primaryKey" json:"id"`
	Name                 string      `gorm:"size:180;not null" json:"name"`
	OldAnchorID          uint        `gorm:"not null;index" json:"old_anchor_id"`
	NewAnchorID          uint        `gorm:"not null;index" json:"new_anchor_id"`
	OldAnchor            TrustAnchor `gorm:"foreignKey:OldAnchorID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"old_anchor"`
	NewAnchor            TrustAnchor `gorm:"foreignKey:NewAnchorID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"new_anchor"`
	OverlapStart         time.Time   `gorm:"not null" json:"overlap_start"`
	OverlapEnd           time.Time   `gorm:"not null" json:"overlap_end"`
	CandidateChainIDs    string      `gorm:"type:text;not null" json:"candidate_chain_ids"`
	AlgorithmVersion     string      `gorm:"size:80;not null;uniqueIndex:idx_scenario_input,priority:2" json:"algorithm_version"`
	InputHash            string      `gorm:"size:64;not null;uniqueIndex:idx_scenario_input,priority:1" json:"input_hash"`
	InputSnapshot        string      `gorm:"type:text;not null" json:"-"`
	SimulationTime       time.Time   `gorm:"not null;uniqueIndex:idx_scenario_input,priority:3" json:"simulation_time"`
	AffectedServicesJSON string      `gorm:"type:text;not null" json:"affected_services_json"`
	BrokenPathsJSON      string      `gorm:"type:text;not null" json:"broken_paths_json"`
	PathEvidenceJSON     string      `gorm:"type:text;not null" json:"path_evidence_json"`
	ScenarioState        string      `gorm:"size:24;not null;index" json:"scenario_state"`
	Explanation          string      `gorm:"type:text;not null" json:"explanation"`
	CreatedBy            uint        `gorm:"not null;index" json:"created_by"`
	CreatedByName        string      `gorm:"size:80;not null" json:"created_by_name"`
	VerifiedBy           *uint       `json:"verified_by,omitempty"`
	VerifiedByName       string      `gorm:"size:80" json:"verified_by_name"`
	IdempotencyKey       string      `gorm:"uniqueIndex;size:128" json:"idempotency_key"`
	ReplayVerified       bool        `gorm:"not null;default:false" json:"replay_verified"`
	DurationMS           int64       `gorm:"not null;default:0" json:"duration_ms"`
	RollbackRecord       string      `gorm:"type:text" json:"rollback_record"`
	CreatedAt            time.Time   `gorm:"not null" json:"created_at"`
	UpdatedAt            time.Time   `gorm:"not null" json:"updated_at"`
}

func (s RolloverScenario) ReviewerSeparated(userID uint) bool { return s.CreatedBy != userID }
