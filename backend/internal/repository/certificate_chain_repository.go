package repository

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"pki-certificate-rollover-impact/backend/internal/dto"
	"pki-certificate-rollover-impact/backend/internal/model"
	"strings"
	"time"
)

type CertificateChainRepository interface {
	Create(context.Context, *model.CertificateChain) error
	GetByID(context.Context, uint, bool) (model.CertificateChain, error)
	List(context.Context, dto.CertificateChainQuery) ([]model.CertificateChain, int64, error)
	Transition(context.Context, uint, string, string) (bool, error)
	CountServices(context.Context, uint) (int64, error)
	GetByIDs(context.Context, []uint) ([]model.CertificateChain, error)
}
type certificateChainRepository struct{ db *gorm.DB }

func NewCertificateChainRepository(db *gorm.DB) CertificateChainRepository {
	return &certificateChainRepository{db: db}
}
func (r *certificateChainRepository) Create(ctx context.Context, chain *model.CertificateChain) error {
	if err := scopedDB(ctx, r.db).Create(chain).Error; err != nil {
		return fmt.Errorf("create certificate chain: %w", err)
	}
	return nil
}
func (r *certificateChainRepository) GetByID(ctx context.Context, id uint, preload bool) (model.CertificateChain, error) {
	var chain model.CertificateChain
	query := scopedDB(ctx, r.db)
	if preload {
		query = query.Preload("TrustAnchor")
	}
	if err := query.First(&chain, id).Error; err != nil {
		return model.CertificateChain{}, fmt.Errorf("find certificate chain %d: %w", id, err)
	}
	return chain, nil
}
func (r *certificateChainRepository) List(ctx context.Context, query dto.CertificateChainQuery) ([]model.CertificateChain, int64, error) {
	base := scopedDB(ctx, r.db).Model(&model.CertificateChain{})
	if query.Search != "" {
		pattern := "%" + strings.ToLower(query.Search) + "%"
		base = base.Where("LOWER(chain_code) LIKE ? OR LOWER(leaf_subject) LIKE ?", pattern, pattern)
	}
	if query.TrustAnchorID != 0 {
		base = base.Where("trust_anchor_id = ?", query.TrustAnchorID)
	}
	if query.State != "" {
		base = base.Where("chain_state = ?", query.State)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count certificate chains: %w", err)
	}
	var chains []model.CertificateChain
	offset := (query.Page - 1) * query.PageSize
	if err := base.Preload("TrustAnchor").Order("valid_to ASC, id DESC").Limit(query.PageSize).Offset(offset).Find(&chains).Error; err != nil {
		return nil, 0, fmt.Errorf("list certificate chains: %w", err)
	}
	return chains, total, nil
}
func (r *certificateChainRepository) Transition(ctx context.Context, id uint, from, to string) (bool, error) {
	result := scopedDB(ctx, r.db).Model(&model.CertificateChain{}).Where("id = ? AND chain_state = ?", id, from).Updates(map[string]any{"chain_state": to, "updated_at": time.Now().UTC()})
	if result.Error != nil {
		return false, fmt.Errorf("transition certificate chain: %w", result.Error)
	}
	return result.RowsAffected == 1, nil
}
func (r *certificateChainRepository) CountServices(ctx context.Context, id uint) (int64, error) {
	var count int64
	if err := scopedDB(ctx, r.db).Model(&model.DependentService{}).Where("chain_id = ?", id).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count chain services: %w", err)
	}
	return count, nil
}
func (r *certificateChainRepository) GetByIDs(ctx context.Context, ids []uint) ([]model.CertificateChain, error) {
	var chains []model.CertificateChain
	if err := scopedDB(ctx, r.db).Preload("TrustAnchor").Where("id IN ?", ids).Find(&chains).Error; err != nil {
		return nil, fmt.Errorf("find candidate chains: %w", err)
	}
	return chains, nil
}
