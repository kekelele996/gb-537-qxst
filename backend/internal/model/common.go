package model

import "time"

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:80;not null" json:"username"`
	DisplayName  string    `gorm:"size:120;not null" json:"display_name"`
	Team         string    `gorm:"size:160;not null" json:"team"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Role         string    `gorm:"size:40;not null;index" json:"role"`
	Active       bool      `gorm:"not null;default:true" json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type AuditLog struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	RequestID        string     `gorm:"size:64;not null;index" json:"request_id"`
	ActorID          uint       `gorm:"not null;index" json:"actor_id"`
	ActorName        string     `gorm:"size:80;not null" json:"actor_name"`
	ActorRole        string     `gorm:"size:40;not null" json:"actor_role"`
	EntityType       string     `gorm:"size:80;not null;index" json:"entity_type"`
	EntityID         uint       `gorm:"not null;index" json:"entity_id"`
	Action           string     `gorm:"size:80;not null;index" json:"action"`
	BeforeSnapshot   string     `gorm:"type:text;not null;default:'{}'" json:"before_snapshot"`
	AfterSnapshot    string     `gorm:"type:text;not null;default:'{}'" json:"after_snapshot"`
	InputHash        string     `gorm:"size:64" json:"input_hash"`
	AlgorithmVersion string     `gorm:"size:80" json:"algorithm_version"`
	SimulationTime   *time.Time `json:"simulation_time,omitempty"`
	DurationMS       int64      `gorm:"not null;default:0" json:"duration_ms"`
	PathSummary      string     `gorm:"type:text" json:"path_summary"`
	CreatedAt        time.Time  `gorm:"index;not null" json:"created_at"`
}
