package model

import "time"

type DependentService struct {
	ID                  uint             `gorm:"primaryKey" json:"id"`
	ServiceCode         string           `gorm:"uniqueIndex;size:80;not null" json:"service_code"`
	Name                string           `gorm:"size:160;not null" json:"name"`
	OwnerTeam           string           `gorm:"size:160;not null;index" json:"owner_team"`
	Environment         string           `gorm:"size:40;not null;index" json:"environment"`
	ChainID             uint             `gorm:"not null;index" json:"chain_id"`
	Chain               CertificateChain `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"chain"`
	ClientTrustRefsJSON string           `gorm:"type:text;not null" json:"client_trust_refs_json"`
	Protocol            string           `gorm:"size:40;not null" json:"protocol"`
	Criticality         string           `gorm:"size:24;not null;index" json:"criticality"`
	DependencyEdgesJSON string           `gorm:"type:text;not null" json:"dependency_edges_json"`
	ServiceState        string           `gorm:"size:24;not null;index" json:"service_state"`
	CreatedAt           time.Time        `gorm:"not null" json:"created_at"`
	UpdatedAt           time.Time        `gorm:"not null" json:"updated_at"`
}
