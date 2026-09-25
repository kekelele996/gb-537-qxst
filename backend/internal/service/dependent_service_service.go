package service

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"net/http"
	"pki-certificate-rollover-impact/backend/internal/constants"
	"pki-certificate-rollover-impact/backend/internal/dto"
	"pki-certificate-rollover-impact/backend/internal/model"
	"pki-certificate-rollover-impact/backend/internal/repository"
	"pki-certificate-rollover-impact/backend/internal/util"
	"time"
)

type DependentServiceService struct {
	services     repository.DependentServiceRepository
	chains       repository.CertificateChainRepository
	anchors      repository.TrustAnchorRepository
	audits       repository.AuditRepository
	transactions repository.TransactionManager
	now          func() time.Time
}

func NewDependentServiceService(services repository.DependentServiceRepository, chains repository.CertificateChainRepository, anchors repository.TrustAnchorRepository, audits repository.AuditRepository, transactions repository.TransactionManager) *DependentServiceService {
	return &DependentServiceService{services: services, chains: chains, anchors: anchors, audits: audits, transactions: transactions, now: func() time.Time { return time.Now().UTC() }}
}

func requireServiceOwnership(actor util.Actor, ownerTeam string) error {
	if actor.Role == string(constants.RoleAdmin) || actor.Role == string(constants.RolePKIOperator) {
		return nil
	}
	if actor.Role == string(constants.RoleServiceOwner) && actor.Team == ownerTeam {
		return nil
	}
	return util.NewError(http.StatusForbidden, util.CodeForbidden, "service owner may only modify services owned by their team")
}
func (s *DependentServiceService) validateReferences(ctx context.Context, serviceID, chainID uint, trustIDs, edgeIDs []uint) error {
	if _, err := s.chains.GetByID(ctx, chainID, false); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return util.NotFound("certificate chain")
		}
		return err
	}
	anchors, err := s.anchors.GetByIDs(ctx, uniqueIDs(trustIDs))
	if err != nil {
		return err
	}
	if len(anchors) != len(uniqueIDs(trustIDs)) {
		return util.NewError(http.StatusBadRequest, util.CodeValidation, "one or more client trust anchors do not exist")
	}
	all, err := s.services.All(ctx)
	if err != nil {
		return err
	}
	known := map[uint]bool{}
	for _, item := range all {
		known[item.ID] = true
	}
	for _, edgeID := range uniqueIDs(edgeIDs) {
		if edgeID == serviceID && serviceID != 0 {
			return util.NewError(http.StatusConflict, util.CodeConflict, "service cannot depend on itself")
		}
		if !known[edgeID] {
			return util.NewError(http.StatusBadRequest, util.CodeValidation, "dependency service does not exist")
		}
	}
	return nil
}
func (s *DependentServiceService) Create(ctx context.Context, request dto.CreateDependentServiceRequest, actor util.Actor, requestID string) (dto.DependentServiceResponse, error) {
	if err := validateRequest(request); err != nil {
		return dto.DependentServiceResponse{}, err
	}
	if err := requireServiceOwnership(actor, request.OwnerTeam); err != nil {
		return dto.DependentServiceResponse{}, err
	}
	if err := s.validateReferences(ctx, 0, request.ChainID, request.ClientTrustRefsJSON, request.DependencyEdgesJSON); err != nil {
		return dto.DependentServiceResponse{}, err
	}
	trustJSON, _ := encode(uniqueIDs(request.ClientTrustRefsJSON))
	edgesJSON, _ := encode(uniqueIDs(request.DependencyEdgesJSON))
	now := s.now()
	service := model.DependentService{ServiceCode: request.ServiceCode, Name: request.Name, OwnerTeam: request.OwnerTeam, Environment: request.Environment, ChainID: request.ChainID, ClientTrustRefsJSON: trustJSON, Protocol: request.Protocol, Criticality: request.Criticality, DependencyEdgesJSON: edgesJSON, ServiceState: string(constants.ServiceActive), CreatedAt: now, UpdatedAt: now}
	all, _ := s.services.All(ctx)
	candidate := append(all, service)
	candidate[len(candidate)-1].ID = ^uint(0)
	if err := servicesContainCycle(candidate); err != nil {
		return dto.DependentServiceResponse{}, util.WrapError(http.StatusConflict, util.CodeConflict, "dependency graph would be cyclic", err)
	}
	err := s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		if createErr := s.services.Create(txCtx, &service); createErr != nil {
			return createErr
		}
		return recordAudit(txCtx, s.audits, actor, requestID, "dependent_service", service.ID, "create", nil, service, "", "", nil, 0, "")
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return dto.DependentServiceResponse{}, util.WrapError(http.StatusConflict, util.CodeConflict, "service code already exists", err)
		}
		return dto.DependentServiceResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to create dependent service", err)
	}
	return s.Get(ctx, service.ID)
}
func (s *DependentServiceService) Get(ctx context.Context, id uint) (dto.DependentServiceResponse, error) {
	service, err := s.services.GetByID(ctx, id, true)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.DependentServiceResponse{}, util.NotFound("dependent service")
		}
		return dto.DependentServiceResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load dependent service", err)
	}
	return dto.NewDependentServiceResponse(service, s.now()), nil
}
func (s *DependentServiceService) List(ctx context.Context, query dto.DependentServiceQuery) (dto.DependentServiceListResponse, error) {
	services, total, err := s.services.List(ctx, query)
	if err != nil {
		return dto.DependentServiceListResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to list dependent services", err)
	}
	response := dto.DependentServiceListResponse{Items: make([]dto.DependentServiceResponse, 0, len(services)), Total: total, Page: query.Page, Size: query.PageSize}
	for _, service := range services {
		response.Items = append(response.Items, dto.NewDependentServiceResponse(service, s.now()))
	}
	return response, nil
}
func (s *DependentServiceService) Update(ctx context.Context, id uint, request dto.UpdateDependentServiceRequest, actor util.Actor, requestID string) (dto.DependentServiceResponse, error) {
	if err := validateRequest(request); err != nil {
		return dto.DependentServiceResponse{}, err
	}
	current, err := s.services.GetByID(ctx, id, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.DependentServiceResponse{}, util.NotFound("dependent service")
		}
		return dto.DependentServiceResponse{}, err
	}
	if err := requireServiceOwnership(actor, current.OwnerTeam); err != nil {
		return dto.DependentServiceResponse{}, err
	}
	if actor.Role == string(constants.RoleServiceOwner) && request.OwnerTeam != actor.Team {
		return dto.DependentServiceResponse{}, util.NewError(http.StatusForbidden, util.CodeForbidden, "service owner cannot transfer ownership to another team")
	}
	if err := s.validateReferences(ctx, id, request.ChainID, request.ClientTrustRefsJSON, request.DependencyEdgesJSON); err != nil {
		return dto.DependentServiceResponse{}, err
	}
	trustJSON, _ := encode(uniqueIDs(request.ClientTrustRefsJSON))
	edgesJSON, _ := encode(uniqueIDs(request.DependencyEdgesJSON))
	all, _ := s.services.All(ctx)
	for index := range all {
		if all[index].ID == id {
			all[index].ChainID = request.ChainID
			all[index].ClientTrustRefsJSON = trustJSON
			all[index].DependencyEdgesJSON = edgesJSON
		}
	}
	if err := servicesContainCycle(all); err != nil {
		return dto.DependentServiceResponse{}, util.WrapError(http.StatusConflict, util.CodeConflict, "dependency graph would be cyclic", err)
	}
	updates := map[string]any{"name": request.Name, "owner_team": request.OwnerTeam, "environment": request.Environment, "chain_id": request.ChainID, "client_trust_refs_json": trustJSON, "protocol": request.Protocol, "criticality": request.Criticality, "dependency_edges_json": edgesJSON}
	after := current
	after.Name = request.Name
	after.OwnerTeam = request.OwnerTeam
	after.Environment = request.Environment
	after.ChainID = request.ChainID
	after.ClientTrustRefsJSON = trustJSON
	after.Protocol = request.Protocol
	after.Criticality = request.Criticality
	after.DependencyEdgesJSON = edgesJSON
	err = s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		if updateErr := s.services.Update(txCtx, id, updates); updateErr != nil {
			return updateErr
		}
		return recordAudit(txCtx, s.audits, actor, requestID, "dependent_service", id, "update", current, after, "", "", nil, 0, "")
	})
	if err != nil {
		return dto.DependentServiceResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to update dependent service", err)
	}
	return s.Get(ctx, id)
}
func (s *DependentServiceService) Deactivate(ctx context.Context, id uint, actor util.Actor, requestID string) (dto.DependentServiceResponse, error) {
	current, err := s.services.GetByID(ctx, id, false)
	if err != nil {
		return dto.DependentServiceResponse{}, util.NotFound("dependent service")
	}
	if err := requireServiceOwnership(actor, current.OwnerTeam); err != nil {
		return dto.DependentServiceResponse{}, err
	}
	if current.ServiceState != string(constants.ServiceActive) {
		return dto.DependentServiceResponse{}, util.NewError(http.StatusConflict, util.CodeConflict, "dependent service is already inactive")
	}
	after := current
	after.ServiceState = string(constants.ServiceInactive)
	err = s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		changed, deactivateErr := s.services.Deactivate(txCtx, id)
		if deactivateErr != nil {
			return deactivateErr
		}
		if !changed {
			return util.NewError(http.StatusConflict, util.CodeConflict, "service state changed concurrently")
		}
		return recordAudit(txCtx, s.audits, actor, requestID, "dependent_service", id, "deactivate", current, after, "", "", nil, 0, "")
	})
	if err != nil {
		return dto.DependentServiceResponse{}, err
	}
	return s.Get(ctx, id)
}
