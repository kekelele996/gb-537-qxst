package dto

import (
	"strings"
	"time"

	"pki-certificate-rollover-impact/backend/internal/constants"
	"pki-certificate-rollover-impact/backend/internal/model"
)

type ImportTrustAnchorRequest struct {
	AnchorCode     string `json:"anchor_code" validate:"required,min=3,max=80"`
	CertificatePEM string `json:"certificate_pem" validate:"required,min=80,max=20000"`
}
type TrustAnchorLifecycleRequest struct {
	Action string `json:"action" validate:"required,oneof=revoke archive restore"`
}
type TrustAnchorQuery struct {
	Search   string
	State    string
	Archived *bool
	Page     int
	PageSize int
}

type TrustAnchorResponse struct {
	ID                uint       `json:"id"`
	AnchorCode        string     `json:"anchor_code"`
	SubjectDN         string     `json:"subject_dn"`
	SerialNumber      string     `json:"serial_number"`
	FingerprintSHA256 string     `json:"fingerprint_sha256"`
	NotBefore         time.Time  `json:"not_before"`
	NotAfter          time.Time  `json:"not_after"`
	KeyAlgorithm      string     `json:"key_algorithm"`
	CertificateState  string     `json:"certificate_state"`
	PemRedacted       string     `json:"pem_redacted"`
	RevokedAt         *time.Time `json:"revoked_at,omitempty"`
	Archived          bool       `json:"archived"`
	ChainCount        int64      `json:"chain_count"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}
type TrustAnchorListResponse struct {
	Items []TrustAnchorResponse `json:"items"`
	Total int64                 `json:"total"`
	Page  int                   `json:"page"`
	Size  int                   `json:"size"`
}

func NewTrustAnchorResponse(anchor model.TrustAnchor, chainCount int64, now time.Time) TrustAnchorResponse {
	state := constants.CalculateCertificateState(now, anchor.NotAfter, anchor.RevokedAt != nil)
	fingerprint := anchor.FingerprintSHA256
	if len(fingerprint) > 16 {
		fingerprint = fingerprint[:16] + "…"
	}
	return TrustAnchorResponse{ID: anchor.ID, AnchorCode: anchor.AnchorCode, SubjectDN: anchor.SubjectDN, SerialNumber: anchor.SerialNumber, FingerprintSHA256: anchor.FingerprintSHA256, NotBefore: anchor.NotBefore, NotAfter: anchor.NotAfter, KeyAlgorithm: anchor.KeyAlgorithm, CertificateState: string(state), PemRedacted: "PUBLIC CERTIFICATE · SHA256 " + strings.ToUpper(fingerprint), RevokedAt: anchor.RevokedAt, Archived: anchor.Archived, ChainCount: chainCount, CreatedAt: anchor.CreatedAt, UpdatedAt: anchor.UpdatedAt}
}
