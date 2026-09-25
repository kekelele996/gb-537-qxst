package model

import "time"

type TrustAnchor struct {
	ID                uint               `gorm:"primaryKey" json:"id"`
	AnchorCode        string             `gorm:"uniqueIndex;size:80;not null" json:"anchor_code"`
	SubjectDN         string             `gorm:"size:500;not null" json:"subject_dn"`
	SerialNumber      string             `gorm:"size:160;not null" json:"serial_number"`
	FingerprintSHA256 string             `gorm:"uniqueIndex;size:64;not null" json:"fingerprint_sha256"`
	NotBefore         time.Time          `gorm:"not null" json:"not_before"`
	NotAfter          time.Time          `gorm:"not null;index" json:"not_after"`
	KeyAlgorithm      string             `gorm:"size:80;not null" json:"key_algorithm"`
	CertificateState  string             `gorm:"size:24;not null;index" json:"certificate_state"`
	PemRedacted       string             `gorm:"type:text;not null" json:"-"`
	RevokedAt         *time.Time         `json:"revoked_at,omitempty"`
	Archived          bool               `gorm:"not null;default:false;index" json:"archived"`
	CreatedAt         time.Time          `gorm:"not null" json:"created_at"`
	UpdatedAt         time.Time          `gorm:"not null" json:"updated_at"`
	Chains            []CertificateChain `gorm:"foreignKey:TrustAnchorID" json:"-"`
}
