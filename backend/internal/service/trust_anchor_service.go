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
	"pki-certificate-rollover-impact/backend/internal/x509util"
	"strings"
	"time"
)

type TrustAnchorService struct {
	anchors      repository.TrustAnchorRepository
	audits       repository.AuditRepository
	transactions repository.TransactionManager
	now          func() time.Time
}

func NewTrustAnchorService(anchors repository.TrustAnchorRepository, audits repository.AuditRepository, transactions repository.TransactionManager) *TrustAnchorService {
	return &TrustAnchorService{anchors: anchors, audits: audits, transactions: transactions, now: func() time.Time { return time.Now().UTC() }}
}
func (s *TrustAnchorService) Import(ctx context.Context, request dto.ImportTrustAnchorRequest, actor util.Actor, requestID string) (dto.TrustAnchorResponse, error) {
	if err := validateRequest(request); err != nil {
		return dto.TrustAnchorResponse{}, err
	}
	certificate, err := x509util.ParseCertificatePEM(request.CertificatePEM)
	if err != nil {
		return dto.TrustAnchorResponse{}, util.WrapError(http.StatusUnprocessableEntity, util.CodeValidation, "public trust-anchor certificate is invalid", err)
	}
	now := s.now()
	if err := x509util.ValidateTrustAnchor(certificate, now); err != nil {
		return dto.TrustAnchorResponse{}, util.WrapError(http.StatusUnprocessableEntity, util.CodeValidation, "trust-anchor validation failed", err)
	}
	anchor := model.TrustAnchor{AnchorCode: strings.TrimSpace(request.AnchorCode), SubjectDN: x509util.SubjectDN(certificate), SerialNumber: certificate.SerialNumber.String(), FingerprintSHA256: x509util.Fingerprint(certificate), NotBefore: certificate.NotBefore.UTC(), NotAfter: certificate.NotAfter.UTC(), KeyAlgorithm: x509util.KeyAlgorithm(certificate), CertificateState: string(constants.CalculateCertificateState(now, certificate.NotAfter, false)), PemRedacted: x509util.NormalizePEM(certificate), CreatedAt: now, UpdatedAt: now}
	err = s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		if createErr := s.anchors.Create(txCtx, &anchor); createErr != nil {
			return createErr
		}
		return recordAudit(txCtx, s.audits, actor, requestID, "trust_anchor", anchor.ID, "import_public_certificate", nil, anchor, "", "", nil, 0, "public certificate fingerprint verified; no private key accepted")
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return dto.TrustAnchorResponse{}, util.WrapError(http.StatusConflict, util.CodeConflict, "anchor code or fingerprint already exists", err)
		}
		var apiErr *util.APIError
		if errors.As(err, &apiErr) {
			return dto.TrustAnchorResponse{}, apiErr
		}
		return dto.TrustAnchorResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to import trust anchor", err)
	}
	return dto.NewTrustAnchorResponse(anchor, 0, now), nil
}
func (s *TrustAnchorService) Get(ctx context.Context, id uint) (dto.TrustAnchorResponse, error) {
	anchor, err := s.anchors.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.TrustAnchorResponse{}, util.NotFound("trust anchor")
		}
		return dto.TrustAnchorResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load trust anchor", err)
	}
	count, err := s.anchors.CountChains(ctx, id)
	if err != nil {
		return dto.TrustAnchorResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to count certificate chains", err)
	}
	return dto.NewTrustAnchorResponse(anchor, count, s.now()), nil
}
func (s *TrustAnchorService) List(ctx context.Context, query dto.TrustAnchorQuery) (dto.TrustAnchorListResponse, error) {
	anchors, total, err := s.anchors.List(ctx, query)
	if err != nil {
		return dto.TrustAnchorListResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to list trust anchors", err)
	}
	response := dto.TrustAnchorListResponse{Items: make([]dto.TrustAnchorResponse, 0, len(anchors)), Total: total, Page: query.Page, Size: query.PageSize}
	for _, anchor := range anchors {
		count, _ := s.anchors.CountChains(ctx, anchor.ID)
		response.Items = append(response.Items, dto.NewTrustAnchorResponse(anchor, count, s.now()))
	}
	return response, nil
}
func (s *TrustAnchorService) Lifecycle(ctx context.Context, id uint, request dto.TrustAnchorLifecycleRequest, actor util.Actor, requestID string) (dto.TrustAnchorResponse, error) {
	if err := validateRequest(request); err != nil {
		return dto.TrustAnchorResponse{}, err
	}
	anchor, err := s.anchors.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.TrustAnchorResponse{}, util.NotFound("trust anchor")
		}
		return dto.TrustAnchorResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load trust anchor", err)
	}
	before := anchor
	updates := map[string]any{}
	now := s.now()
	switch request.Action {
	case "revoke":
		if anchor.RevokedAt != nil {
			return dto.TrustAnchorResponse{}, util.NewError(http.StatusConflict, util.CodeConflict, "trust anchor is already revoked")
		}
		updates["revoked_at"] = now
		updates["certificate_state"] = string(constants.CertificateRevoked)
		anchor.RevokedAt = &now
		anchor.CertificateState = string(constants.CertificateRevoked)
	case "archive":
		updates["archived"] = true
		anchor.Archived = true
	case "restore":
		if anchor.RevokedAt != nil {
			return dto.TrustAnchorResponse{}, util.NewError(http.StatusConflict, util.CodeConflict, "revoked trust anchors cannot be restored")
		}
		updates["archived"] = false
		anchor.Archived = false
	}
	err = s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		if updateErr := s.anchors.SetLifecycle(txCtx, id, updates); updateErr != nil {
			return updateErr
		}
		return recordAudit(txCtx, s.audits, actor, requestID, "trust_anchor", id, request.Action, before, anchor, "", "", nil, 0, "")
	})
	if err != nil {
		return dto.TrustAnchorResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to update trust anchor lifecycle", err)
	}
	return s.Get(ctx, id)
}
