package dto

import (
	"encoding/json"
	"time"

	"pki-certificate-rollover-impact/backend/internal/model"
	"pki-certificate-rollover-impact/backend/internal/x509util"
)

type ImportCertificateChainRequest struct {
	ChainCode       string   `json:"chain_code" validate:"required,min=3,max=80"`
	TrustAnchorID   uint     `json:"trust_anchor_id" validate:"required"`
	CertificatesPEM []string `json:"certificates_pem" validate:"required,min=1,max=8,dive,required"`
}
type CertificateChainTransitionRequest struct {
	ToState string `json:"to_state" validate:"required,oneof=validated deprecated revoked"`
}
type CertificateChainQuery struct {
	Search        string
	TrustAnchorID uint
	State         string
	Page          int
	PageSize      int
}

type CertificateChainResponse struct {
	ID                  uint                      `json:"id"`
	ChainCode           string                    `json:"chain_code"`
	TrustAnchorID       uint                      `json:"trust_anchor_id"`
	TrustAnchor         *TrustAnchorResponse      `json:"trust_anchor,omitempty"`
	LeafSubject         string                    `json:"leaf_subject"`
	CertificateRefsJSON []x509util.CertificateRef `json:"certificate_refs_json"`
	ChainFingerprint    string                    `json:"chain_fingerprint"`
	ValidFrom           time.Time                 `json:"valid_from"`
	ValidTo             time.Time                 `json:"valid_to"`
	ValidationResult    x509util.ChainEvidence    `json:"validation_result"`
	ChainState          string                    `json:"chain_state"`
	SourceChecksum      string                    `json:"source_checksum"`
	ServiceCount        int64                     `json:"service_count"`
	CreatedAt           time.Time                 `json:"created_at"`
	UpdatedAt           time.Time                 `json:"updated_at"`
}
type CertificateChainListResponse struct {
	Items []CertificateChainResponse `json:"items"`
	Total int64                      `json:"total"`
	Page  int                        `json:"page"`
	Size  int                        `json:"size"`
}

func NewCertificateChainResponse(chain model.CertificateChain, serviceCount int64, now time.Time) CertificateChainResponse {
	refs := []x509util.CertificateRef{}
	_ = json.Unmarshal([]byte(chain.CertificateRefsJSON), &refs)
	evidence := x509util.ChainEvidence{}
	_ = json.Unmarshal([]byte(chain.ValidationResult), &evidence)
	response := CertificateChainResponse{ID: chain.ID, ChainCode: chain.ChainCode, TrustAnchorID: chain.TrustAnchorID, LeafSubject: chain.LeafSubject, CertificateRefsJSON: refs, ChainFingerprint: chain.ChainFingerprint, ValidFrom: chain.ValidFrom, ValidTo: chain.ValidTo, ValidationResult: evidence, ChainState: chain.ChainState, SourceChecksum: chain.SourceChecksum, ServiceCount: serviceCount, CreatedAt: chain.CreatedAt, UpdatedAt: chain.UpdatedAt}
	if chain.TrustAnchor.ID != 0 {
		anchor := NewTrustAnchorResponse(chain.TrustAnchor, 0, now)
		response.TrustAnchor = &anchor
	}
	return response
}
