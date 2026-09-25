package repository

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"pki-certificate-rollover-impact/backend/internal/dto"
	"pki-certificate-rollover-impact/backend/internal/model"
	"time"
)

type RolloverScenarioRepository interface {
	Create(context.Context, *model.RolloverScenario) error
	GetByID(context.Context, uint, bool) (model.RolloverScenario, error)
	List(context.Context, dto.RolloverScenarioQuery) ([]model.RolloverScenario, int64, error)
	FindByIdempotencyKey(context.Context, string) (model.RolloverScenario, error)
	FindByInput(context.Context, string, string, time.Time) (model.RolloverScenario, error)
	CompleteSimulation(context.Context, uint, map[string]any) (bool, error)
	Transition(context.Context, uint, string, string, map[string]any) (bool, error)
	SetReplayVerified(context.Context, uint, bool) error
}
type rolloverScenarioRepository struct{ db *gorm.DB }

func NewRolloverScenarioRepository(db *gorm.DB) RolloverScenarioRepository {
	return &rolloverScenarioRepository{db: db}
}
func (r *rolloverScenarioRepository) Create(ctx context.Context, scenario *model.RolloverScenario) error {
	if err := scopedDB(ctx, r.db).Create(scenario).Error; err != nil {
		return fmt.Errorf("create rollover scenario: %w", err)
	}
	return nil
}
func (r *rolloverScenarioRepository) GetByID(ctx context.Context, id uint, preload bool) (model.RolloverScenario, error) {
	var scenario model.RolloverScenario
	query := scopedDB(ctx, r.db)
	if preload {
		query = query.Preload("OldAnchor").Preload("NewAnchor")
	}
	if err := query.First(&scenario, id).Error; err != nil {
		return model.RolloverScenario{}, fmt.Errorf("find rollover scenario %d: %w", id, err)
	}
	return scenario, nil
}
func (r *rolloverScenarioRepository) List(ctx context.Context, query dto.RolloverScenarioQuery) ([]model.RolloverScenario, int64, error) {
	base := scopedDB(ctx, r.db).Model(&model.RolloverScenario{})
	if query.State != "" {
		base = base.Where("scenario_state = ?", query.State)
	}
	if query.CreatedBy != 0 {
		base = base.Where("created_by = ?", query.CreatedBy)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count rollover scenarios: %w", err)
	}
	var scenarios []model.RolloverScenario
	offset := (query.Page - 1) * query.PageSize
	if err := base.Preload("OldAnchor").Preload("NewAnchor").Order("created_at DESC, id DESC").Limit(query.PageSize).Offset(offset).Find(&scenarios).Error; err != nil {
		return nil, 0, fmt.Errorf("list rollover scenarios: %w", err)
	}
	return scenarios, total, nil
}
func (r *rolloverScenarioRepository) FindByIdempotencyKey(ctx context.Context, key string) (model.RolloverScenario, error) {
	var scenario model.RolloverScenario
	if err := scopedDB(ctx, r.db).Where("idempotency_key = ?", key).First(&scenario).Error; err != nil {
		return model.RolloverScenario{}, fmt.Errorf("find scenario by idempotency key: %w", err)
	}
	return scenario, nil
}
func (r *rolloverScenarioRepository) FindByInput(ctx context.Context, hash, version string, simulationTime time.Time) (model.RolloverScenario, error) {
	var scenario model.RolloverScenario
	if err := scopedDB(ctx, r.db).Where("input_hash = ? AND algorithm_version = ? AND simulation_time = ?", hash, version, simulationTime).First(&scenario).Error; err != nil {
		return model.RolloverScenario{}, fmt.Errorf("find scenario by frozen input: %w", err)
	}
	return scenario, nil
}
func (r *rolloverScenarioRepository) CompleteSimulation(ctx context.Context, id uint, updates map[string]any) (bool, error) {
	updates["scenario_state"] = "simulated"
	updates["updated_at"] = time.Now().UTC()
	result := scopedDB(ctx, r.db).Model(&model.RolloverScenario{}).Where("id = ? AND scenario_state = ?", id, "draft").Updates(updates)
	if result.Error != nil {
		return false, fmt.Errorf("complete rollover simulation: %w", result.Error)
	}
	return result.RowsAffected == 1, nil
}
func (r *rolloverScenarioRepository) Transition(ctx context.Context, id uint, from, to string, updates map[string]any) (bool, error) {
	if updates == nil {
		updates = map[string]any{}
	}
	updates["scenario_state"] = to
	updates["updated_at"] = time.Now().UTC()
	result := scopedDB(ctx, r.db).Model(&model.RolloverScenario{}).Where("id = ? AND scenario_state = ?", id, from).Updates(updates)
	if result.Error != nil {
		return false, fmt.Errorf("transition rollover scenario: %w", result.Error)
	}
	return result.RowsAffected == 1, nil
}
func (r *rolloverScenarioRepository) SetReplayVerified(ctx context.Context, id uint, passed bool) error {
	result := scopedDB(ctx, r.db).Model(&model.RolloverScenario{}).Where("id = ?", id).Updates(map[string]any{"replay_verified": passed, "updated_at": time.Now().UTC()})
	if result.Error != nil {
		return fmt.Errorf("store replay evidence: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
