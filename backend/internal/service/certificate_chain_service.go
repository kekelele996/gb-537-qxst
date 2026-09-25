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

type CertificateChainService struct {
	chains       repository.CertificateChainRepository
	anchors      repository.TrustAnchorRepository
	audits       repository.AuditRepository
	transactions repository.TransactionManager
	now          func() time.Time
}

func NewCertificateChainService(chains repository.CertificateChainRepository, anchors repository.TrustAnchorRepository, audits repository.AuditRepository, transactions repository.TransactionManager) *CertificateChainService {
	return &CertificateChainService{chains: chains, anchors: anchors, audits: audits, transactions: transactions, now: func() time.Time { return time.Now().UTC() }}
}
func (s *CertificateChainService) Import(ctx context.Context, request dto.ImportCertificateChainRequest, actor util.Actor, requestID string) (dto.CertificateChainResponse, error) {
	if err := validateRequest(request); err != nil {
		return dto.CertificateChainResponse{}, err
	}
	anchor, err := s.anchors.GetByID(ctx, request.TrustAnchorID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.CertificateChainResponse{}, util.NotFound("trust anchor")
		}
		return dto.CertificateChainResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load trust anchor", err)
	}
	bundle := strings.Join(request.CertificatesPEM, "\n")
	evidence, refs, err := x509util.ValidateChain(anchor.PemRedacted, bundle, s.now())
	if err != nil {
		return dto.CertificateChainResponse{}, util.WrapError(http.StatusUnprocessableEntity, util.CodeValidation, "certificate chain failed offline validation", err)
	}
	certificates, err := x509util.ParseCertificateBundle(bundle)
	if err != nil {
		return dto.CertificateChainResponse{}, util.WrapError(http.StatusUnprocessableEntity, util.CodeValidation, "certificate bundle is invalid", err)
	}
	validFrom := anchor.NotBefore
	validTo := anchor.NotAfter
	for _, certificate := range certificates {
		if certificate.NotBefore.After(validFrom) {
			validFrom = certificate.NotBefore
		}
		if certificate.NotAfter.Before(validTo) {
			validTo = certificate.NotAfter
		}
	}
	refsJSON, _ := encode(refs)
	evidenceJSON, _ := encode(evidence)
	fingerprints := make([]string, 0, len(refs))
	for _, ref := range refs {
		fingerprints = append(fingerprints, ref.FingerprintSHA256)
	}
	now := s.now()
	chain := model.CertificateChain{ChainCode: strings.TrimSpace(request.ChainCode), TrustAnchorID: anchor.ID, LeafSubject: certificates[0].Subject.String(), CertificateRefsJSON: refsJSON, ChainFingerprint: util.HashString(strings.Join(fingerprints, ":")), ValidFrom: validFrom.UTC(), ValidTo: validTo.UTC(), ValidationResult: evidenceJSON, ChainState: string(constants.ChainValidated), SourceChecksum: util.HashString(bundle), PublicChainPEM: bundle, CreatedAt: now, UpdatedAt: now}
	err = s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		if createErr := s.chains.Create(txCtx, &chain); createErr != nil {
			return createErr
		}
		return recordAudit(txCtx, s.audits, actor, requestID, "certificate_chain", chain.ID, "import_and_validate", nil, chain, chain.SourceChecksum, "x509-standard-library", nil, 0, evidence.Message)
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return dto.CertificateChainResponse{}, util.WrapError(http.StatusConflict, util.CodeConflict, "chain code or fingerprint already exists", err)
		}
		var apiErr *util.APIError
		if errors.As(err, &apiErr) {
			return dto.CertificateChainResponse{}, apiErr
		}
		return dto.CertificateChainResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to import certificate chain", err)
	}
	chain.TrustAnchor = anchor
	return dto.NewCertificateChainResponse(chain, 0, now), nil
}
func (s *CertificateChainService) Get(ctx context.Context, id uint) (dto.CertificateChainResponse, error) {
	chain, err := s.chains.GetByID(ctx, id, true)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.CertificateChainResponse{}, util.NotFound("certificate chain")
		}
		return dto.CertificateChainResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load certificate chain", err)
	}
	count, err := s.chains.CountServices(ctx, id)
	if err != nil {
		return dto.CertificateChainResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to count dependent services", err)
	}
	return dto.NewCertificateChainResponse(chain, count, s.now()), nil
}
func (s *CertificateChainService) List(ctx context.Context, query dto.CertificateChainQuery) (dto.CertificateChainListResponse, error) {
	chains, total, err := s.chains.List(ctx, query)
	if err != nil {
		return dto.CertificateChainListResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to list certificate chains", err)
	}
	response := dto.CertificateChainListResponse{Items: make([]dto.CertificateChainResponse, 0, len(chains)), Total: total, Page: query.Page, Size: query.PageSize}
	for _, chain := range chains {
		count, _ := s.chains.CountServices(ctx, chain.ID)
		response.Items = append(response.Items, dto.NewCertificateChainResponse(chain, count, s.now()))
	}
	return response, nil
}
func (s *CertificateChainService) Transition(ctx context.Context, id uint, request dto.CertificateChainTransitionRequest, actor util.Actor, requestID string) (dto.CertificateChainResponse, error) {
	if err := validateRequest(request); err != nil {
		return dto.CertificateChainResponse{}, err
	}
	chain, err := s.chains.GetByID(ctx, id, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.CertificateChainResponse{}, util.NotFound("certificate chain")
		}
		return dto.CertificateChainResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load certificate chain", err)
	}
	from, to := constants.ChainState(chain.ChainState), constants.ChainState(request.ToState)
	if !constants.CanTransitionChain(from, to) {
		return dto.CertificateChainResponse{}, util.NewError(http.StatusConflict, util.CodeStateTransition, "illegal chain transition from "+chain.ChainState+" to "+request.ToState)
	}
	before := chain
	err = s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		changed, transitionErr := s.chains.Transition(txCtx, id, chain.ChainState, request.ToState)
		if transitionErr != nil {
			return transitionErr
		}
		if !changed {
			return util.NewError(http.StatusConflict, util.CodeConflict, "certificate chain state changed concurrently")
		}
		chain.ChainState = request.ToState
		return recordAudit(txCtx, s.audits, actor, requestID, "certificate_chain", id, "transition", before, chain, chain.SourceChecksum, "x509-standard-library", nil, 0, "")
	})
	if err != nil {
		var apiErr *util.APIError
		if errors.As(err, &apiErr) {
			return dto.CertificateChainResponse{}, apiErr
		}
		return dto.CertificateChainResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to transition certificate chain", err)
	}
	return s.Get(ctx, id)
}
func (s *CertificateChainService) Compare(ctx context.Context, id, otherID uint) (map[string]any, error) {
	first, err := s.chains.GetByID(ctx, id, true)
	if err != nil {
		return nil, util.NotFound("first certificate chain")
	}
	second, err := s.chains.GetByID(ctx, otherID, true)
	if err != nil {
		return nil, util.NotFound("second certificate chain")
	}
	overlapStart := first.ValidFrom
	if second.ValidFrom.After(overlapStart) {
		overlapStart = second.ValidFrom
	}
	overlapEnd := first.ValidTo
	if second.ValidTo.Before(overlapEnd) {
		overlapEnd = second.ValidTo
	}
	return map[string]any{"first_id": first.ID, "second_id": second.ID, "same_leaf_subject": first.LeafSubject == second.LeafSubject, "anchor_changed": first.TrustAnchorID != second.TrustAnchorID, "overlap_start": overlapStart, "overlap_end": overlapEnd, "has_valid_overlap": overlapStart.Before(overlapEnd), "first_fingerprint": first.ChainFingerprint, "second_fingerprint": second.ChainFingerprint}, nil
}
