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

type DependentServiceRepository interface {
	Create(context.Context, *model.DependentService) error
	GetByID(context.Context, uint, bool) (model.DependentService, error)
	List(context.Context, dto.DependentServiceQuery) ([]model.DependentService, int64, error)
	Update(context.Context, uint, map[string]any) error
	Deactivate(context.Context, uint) (bool, error)
	All(context.Context) ([]model.DependentService, error)
}
type dependentServiceRepository struct{ db *gorm.DB }

func NewDependentServiceRepository(db *gorm.DB) DependentServiceRepository {
	return &dependentServiceRepository{db: db}
}
func (r *dependentServiceRepository) Create(ctx context.Context, service *model.DependentService) error {
	if err := scopedDB(ctx, r.db).Create(service).Error; err != nil {
		return fmt.Errorf("create dependent service: %w", err)
	}
	return nil
}
func (r *dependentServiceRepository) GetByID(ctx context.Context, id uint, preload bool) (model.DependentService, error) {
	var service model.DependentService
	query := scopedDB(ctx, r.db)
	if preload {
		query = query.Preload("Chain").Preload("Chain.TrustAnchor")
	}
	if err := query.First(&service, id).Error; err != nil {
		return model.DependentService{}, fmt.Errorf("find dependent service %d: %w", id, err)
	}
	return service, nil
}
func (r *dependentServiceRepository) List(ctx context.Context, query dto.DependentServiceQuery) ([]model.DependentService, int64, error) {
	base := scopedDB(ctx, r.db).Model(&model.DependentService{})
	if query.Search != "" {
		pattern := "%" + strings.ToLower(query.Search) + "%"
		base = base.Where("LOWER(service_code) LIKE ? OR LOWER(name) LIKE ? OR LOWER(owner_team) LIKE ?", pattern, pattern, pattern)
	}
	if query.Environment != "" {
		base = base.Where("environment = ?", query.Environment)
	}
	if query.State != "" {
		base = base.Where("service_state = ?", query.State)
	}
	if query.ChainID != 0 {
		base = base.Where("chain_id = ?", query.ChainID)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count dependent services: %w", err)
	}
	var services []model.DependentService
	offset := (query.Page - 1) * query.PageSize
	if err := base.Preload("Chain").Preload("Chain.TrustAnchor").Order("criticality DESC, service_code ASC").Limit(query.PageSize).Offset(offset).Find(&services).Error; err != nil {
		return nil, 0, fmt.Errorf("list dependent services: %w", err)
	}
	return services, total, nil
}
func (r *dependentServiceRepository) Update(ctx context.Context, id uint, updates map[string]any) error {
	updates["updated_at"] = time.Now().UTC()
	result := scopedDB(ctx, r.db).Model(&model.DependentService{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update dependent service: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func (r *dependentServiceRepository) Deactivate(ctx context.Context, id uint) (bool, error) {
	result := scopedDB(ctx, r.db).Model(&model.DependentService{}).Where("id = ? AND service_state = ?", id, "active").Updates(map[string]any{"service_state": "inactive", "updated_at": time.Now().UTC()})
	if result.Error != nil {
		return false, fmt.Errorf("deactivate dependent service: %w", result.Error)
	}
	return result.RowsAffected == 1, nil
}
func (r *dependentServiceRepository) All(ctx context.Context) ([]model.DependentService, error) {
	var services []model.DependentService
	if err := scopedDB(ctx, r.db).Preload("Chain").Preload("Chain.TrustAnchor").Order("id ASC").Find(&services).Error; err != nil {
		return nil, fmt.Errorf("load dependency graph: %w", err)
	}
	return services, nil
}
