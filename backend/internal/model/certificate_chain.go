package model

import "time"

type CertificateChain struct {
	ID                  uint        `gorm:"primaryKey" json:"id"`
	ChainCode           string      `gorm:"uniqueIndex;size:80;not null" json:"chain_code"`
	TrustAnchorID       uint        `gorm:"not null;index" json:"trust_anchor_id"`
	TrustAnchor         TrustAnchor `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"trust_anchor"`
	LeafSubject         string      `gorm:"size:500;not null;index" json:"leaf_subject"`
	CertificateRefsJSON string      `gorm:"type:text;not null" json:"certificate_refs_json"`
	ChainFingerprint    string      `gorm:"uniqueIndex;size:64;not null" json:"chain_fingerprint"`
	ValidFrom           time.Time   `gorm:"not null" json:"valid_from"`
	ValidTo             time.Time   `gorm:"not null;index" json:"valid_to"`
	ValidationResult    string      `gorm:"type:text;not null" json:"validation_result"`
	ChainState          string      `gorm:"size:24;not null;index" json:"chain_state"`
	SourceChecksum      string      `gorm:"size:64;not null;index" json:"source_checksum"`
	PublicChainPEM      string      `gorm:"type:text;not null" json:"-"`
	CreatedAt           time.Time   `gorm:"not null" json:"created_at"`
	UpdatedAt           time.Time   `gorm:"not null" json:"updated_at"`
}
