package dto

import (
	"encoding/json"
	"time"

	"pki-certificate-rollover-impact/backend/internal/model"
)

type CreateDependentServiceRequest struct {
	ServiceCode         string `json:"service_code" validate:"required,min=3,max=80"`
	Name                string `json:"name" validate:"required,min=2,max=160"`
	OwnerTeam           string `json:"owner_team" validate:"required,min=2,max=160"`
	Environment         string `json:"environment" validate:"required,oneof=development staging production disaster_recovery"`
	ChainID             uint   `json:"chain_id" validate:"required"`
	ClientTrustRefsJSON []uint `json:"client_trust_refs_json" validate:"required,min=1,max=32,dive,gt=0"`
	Protocol            string `json:"protocol" validate:"required,oneof=tls mtls ldaps smtps kafka_tls"`
	Criticality         string `json:"criticality" validate:"required,oneof=low medium high critical"`
	DependencyEdgesJSON []uint `json:"dependency_edges_json" validate:"max=64,dive,gt=0"`
}
type UpdateDependentServiceRequest struct {
	Name                string `json:"name" validate:"required,min=2,max=160"`
	OwnerTeam           string `json:"owner_team" validate:"required,min=2,max=160"`
	Environment         string `json:"environment" validate:"required,oneof=development staging production disaster_recovery"`
	ChainID             uint   `json:"chain_id" validate:"required"`
	ClientTrustRefsJSON []uint `json:"client_trust_refs_json" validate:"required,min=1,max=32,dive,gt=0"`
	Protocol            string `json:"protocol" validate:"required,oneof=tls mtls ldaps smtps kafka_tls"`
	Criticality         string `json:"criticality" validate:"required,oneof=low medium high critical"`
	DependencyEdgesJSON []uint `json:"dependency_edges_json" validate:"max=64,dive,gt=0"`
}
type DependentServiceQuery struct {
	Search      string
	Environment string
	State       string
	ChainID     uint
	Page        int
	PageSize    int
}

type DependentServiceResponse struct {
	ID                  uint                      `json:"id"`
	ServiceCode         string                    `json:"service_code"`
	Name                string                    `json:"name"`
	OwnerTeam           string                    `json:"owner_team"`
	Environment         string                    `json:"environment"`
	ChainID             uint                      `json:"chain_id"`
	Chain               *CertificateChainResponse `json:"chain,omitempty"`
	ClientTrustRefsJSON []uint                    `json:"client_trust_refs_json"`
	Protocol            string                    `json:"protocol"`
	Criticality         string                    `json:"criticality"`
	DependencyEdgesJSON []uint                    `json:"dependency_edges_json"`
	ServiceState        string                    `json:"service_state"`
	CreatedAt           time.Time                 `json:"created_at"`
	UpdatedAt           time.Time                 `json:"updated_at"`
}
type DependentServiceListResponse struct {
	Items []DependentServiceResponse `json:"items"`
	Total int64                      `json:"total"`
	Page  int                        `json:"page"`
	Size  int                        `json:"size"`
}

func NewDependentServiceResponse(service model.DependentService, now time.Time) DependentServiceResponse {
	trust := []uint{}
	edges := []uint{}
	_ = json.Unmarshal([]byte(service.ClientTrustRefsJSON), &trust)
	_ = json.Unmarshal([]byte(service.DependencyEdgesJSON), &edges)
	response := DependentServiceResponse{ID: service.ID, ServiceCode: service.ServiceCode, Name: service.Name, OwnerTeam: service.OwnerTeam, Environment: service.Environment, ChainID: service.ChainID, ClientTrustRefsJSON: trust, Protocol: service.Protocol, Criticality: service.Criticality, DependencyEdgesJSON: edges, ServiceState: service.ServiceState, CreatedAt: service.CreatedAt, UpdatedAt: service.UpdatedAt}
	if service.Chain.ID != 0 {
		chain := NewCertificateChainResponse(service.Chain, 0, now)
		response.Chain = &chain
	}
	return response
}
