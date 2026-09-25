package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"pki-certificate-rollover-impact/backend/internal/dto"
	"pki-certificate-rollover-impact/backend/internal/model"
)

type transactionContextKey struct{}
type TransactionManager interface {
	WithinTransaction(context.Context, func(context.Context) error) error
}
type transactionManager struct{ db *gorm.DB }

func NewTransactionManager(db *gorm.DB) TransactionManager { return &transactionManager{db: db} }
func (manager *transactionManager) WithinTransaction(ctx context.Context, work func(context.Context) error) error {
	return manager.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return work(context.WithValue(ctx, transactionContextKey{}, tx)) })
}
func scopedDB(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(transactionContextKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return db.WithContext(ctx)
}

type UserRepository interface {
	FindByUsername(context.Context, string) (model.User, error)
}
type userRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) UserRepository { return &userRepository{db: db} }
func (r *userRepository) FindByUsername(ctx context.Context, username string) (model.User, error) {
	var user model.User
	if err := scopedDB(ctx, r.db).Where("username = ?", username).First(&user).Error; err != nil {
		return model.User{}, fmt.Errorf("find user by username: %w", err)
	}
	return user, nil
}

type AuditQuery struct {
	EntityType, RequestID, Actor, Action string
	From, To                             *time.Time
	Page, PageSize                       int
}
type AuditRepository interface {
	Record(context.Context, model.AuditLog) error
	List(context.Context, AuditQuery) ([]model.AuditLog, int64, error)
}
type auditRepository struct{ db *gorm.DB }

func NewAuditRepository(db *gorm.DB) AuditRepository { return &auditRepository{db: db} }
func (r *auditRepository) Record(ctx context.Context, entry model.AuditLog) error {
	if err := scopedDB(ctx, r.db).Create(&entry).Error; err != nil {
		return fmt.Errorf("append immutable audit log: %w", err)
	}
	return nil
}
func (r *auditRepository) List(ctx context.Context, query AuditQuery) ([]model.AuditLog, int64, error) {
	base := scopedDB(ctx, r.db).Model(&model.AuditLog{})
	if query.EntityType != "" {
		base = base.Where("entity_type = ?", query.EntityType)
	}
	if query.RequestID != "" {
		base = base.Where("request_id = ?", query.RequestID)
	}
	if query.Actor != "" {
		pattern := "%" + strings.ToLower(query.Actor) + "%"
		base = base.Where("LOWER(actor_name) LIKE ?", pattern)
	}
	if query.Action != "" {
		base = base.Where("action = ?", query.Action)
	}
	if query.From != nil {
		base = base.Where("created_at >= ?", *query.From)
	}
	if query.To != nil {
		base = base.Where("created_at <= ?", *query.To)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}
	var entries []model.AuditLog
	offset := (query.Page - 1) * query.PageSize
	if err := base.Order("created_at DESC, id DESC").Limit(query.PageSize).Offset(offset).Find(&entries).Error; err != nil {
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}
	return entries, total, nil
}

type TrustAnchorRepository interface {
	Create(context.Context, *model.TrustAnchor) error
	GetByID(context.Context, uint) (model.TrustAnchor, error)
	List(context.Context, dto.TrustAnchorQuery) ([]model.TrustAnchor, int64, error)
	SetLifecycle(context.Context, uint, map[string]any) error
	CountChains(context.Context, uint) (int64, error)
	GetByIDs(context.Context, []uint) ([]model.TrustAnchor, error)
}
type trustAnchorRepository struct{ db *gorm.DB }

func NewTrustAnchorRepository(db *gorm.DB) TrustAnchorRepository {
	return &trustAnchorRepository{db: db}
}
func (r *trustAnchorRepository) Create(ctx context.Context, anchor *model.TrustAnchor) error {
	if err := scopedDB(ctx, r.db).Create(anchor).Error; err != nil {
		return fmt.Errorf("create trust anchor: %w", err)
	}
	return nil
}
func (r *trustAnchorRepository) GetByID(ctx context.Context, id uint) (model.TrustAnchor, error) {
	var anchor model.TrustAnchor
	if err := scopedDB(ctx, r.db).First(&anchor, id).Error; err != nil {
		return model.TrustAnchor{}, fmt.Errorf("find trust anchor %d: %w", id, err)
	}
	return anchor, nil
}
func (r *trustAnchorRepository) List(ctx context.Context, query dto.TrustAnchorQuery) ([]model.TrustAnchor, int64, error) {
	base := scopedDB(ctx, r.db).Model(&model.TrustAnchor{})
	if query.Search != "" {
		pattern := "%" + strings.ToLower(query.Search) + "%"
		base = base.Where("LOWER(anchor_code) LIKE ? OR LOWER(subject_dn) LIKE ? OR LOWER(fingerprint_sha256) LIKE ?", pattern, pattern, pattern)
	}
	if query.State != "" {
		base = base.Where("certificate_state = ?", query.State)
	}
	if query.Archived != nil {
		base = base.Where("archived = ?", *query.Archived)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count trust anchors: %w", err)
	}
	var anchors []model.TrustAnchor
	offset := (query.Page - 1) * query.PageSize
	if err := base.Order("not_after ASC, id DESC").Limit(query.PageSize).Offset(offset).Find(&anchors).Error; err != nil {
		return nil, 0, fmt.Errorf("list trust anchors: %w", err)
	}
	return anchors, total, nil
}
func (r *trustAnchorRepository) SetLifecycle(ctx context.Context, id uint, updates map[string]any) error {
	updates["updated_at"] = time.Now().UTC()
	result := scopedDB(ctx, r.db).Model(&model.TrustAnchor{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update trust anchor lifecycle: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func (r *trustAnchorRepository) CountChains(ctx context.Context, id uint) (int64, error) {
	var count int64
	if err := scopedDB(ctx, r.db).Model(&model.CertificateChain{}).Where("trust_anchor_id = ?", id).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count anchor chains: %w", err)
	}
	return count, nil
}
func (r *trustAnchorRepository) GetByIDs(ctx context.Context, ids []uint) ([]model.TrustAnchor, error) {
	var anchors []model.TrustAnchor
	if err := scopedDB(ctx, r.db).Where("id IN ?", ids).Find(&anchors).Error; err != nil {
		return nil, fmt.Errorf("find trust anchors: %w", err)
	}
	return anchors, nil
}
