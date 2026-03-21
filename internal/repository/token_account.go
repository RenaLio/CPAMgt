package repository

import (
	"context"
	v1 "cpamgt/api/v1"
	"errors"

	"cpamgt/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrTokenAccountNotFound  = errors.New("token account not found")
	ErrInvalidTokenAccountID = errors.New("token account id must be greater than zero")
)

const (
	defaultTokenAccountListLimit = 20
	maxTokenAccountListLimit     = 200
)

type TokenAccountListFilter struct {
	Status         *model.TokenAccountStatus
	Email          string
	AccountID      string
	IDToken        string
	IncludeDeleted bool
	Limit          int
	Offset         int
}

type TokenAccountRepository interface {
	Create(ctx context.Context, tokenAccount *model.TokenAccount) error
	CreateByBatch(ctx context.Context, accounts []*model.TokenAccount) int64
	GetByID(ctx context.Context, id uint64, includeDeleted bool) (*model.TokenAccount, error)
	GetByIDToken(ctx context.Context, idToken string, includeDeleted bool) (*model.TokenAccount, error)
	List(ctx context.Context, filter TokenAccountListFilter) ([]model.TokenAccount, int64, error)
	Update(ctx context.Context, tokenAccount *model.TokenAccount) error
	Delete(ctx context.Context, id uint64) error
	Restore(ctx context.Context, id uint64) error
}

type TokenAccountRepo struct {
	*Repository
}

func NewTokenAccountRepo(r *Repository) TokenAccountRepository {
	return &TokenAccountRepo{
		Repository: r,
	}
}

func (r *TokenAccountRepo) Create(ctx context.Context, tokenAccount *model.TokenAccount) error {
	return gorm.G[model.TokenAccount](r.DB(ctx)).Create(ctx, tokenAccount)
}

func (r *TokenAccountRepo) CreateByBatch(ctx context.Context, accounts []*model.TokenAccount) int64 {
	db := r.DB(ctx)
	tx := db.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(accounts, 100)
	//tx := db.Clauses(clause.OnConflict{
	//	Columns: []clause.Column{
	//		{Name: "id_token"},
	//		{Name: "deleted_at"}, // 与唯一索引一致
	//	},
	//	DoUpdates: clause.AssignmentColumns([]string{
	//		"tenant_id",
	//		"access_token",
	//		"refresh_token",
	//		"account_id",
	//		"last_refresh",
	//		"email",
	//		"type",
	//		"expired",
	//		"status",
	//		"percent",
	//		"quota_refresh_time",
	//		"cpa_flag",
	//		"extra",
	//		// 注意：updated_at 会自动更新，无需显式列出
	//	}),
	//	DoNothing: true,
	//}).CreateInBatches(accounts, 100)
	return tx.RowsAffected
}

func (r *TokenAccountRepo) GetByID(ctx context.Context, id uint64, includeDeleted bool) (*model.TokenAccount, error) {
	query := r.baseQuery(ctx, includeDeleted)

	var tokenAccount model.TokenAccount
	err := query.First(&tokenAccount, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, v1.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &tokenAccount, nil
}

func (r *TokenAccountRepo) GetByIDToken(ctx context.Context, idToken string, includeDeleted bool) (*model.TokenAccount, error) {
	query := r.baseQuery(ctx, includeDeleted)
	var tokenAccount model.TokenAccount
	err := query.Where("id_token = ?", idToken).First(&tokenAccount).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, v1.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &tokenAccount, nil
}

func (r *TokenAccountRepo) List(ctx context.Context, filter TokenAccountListFilter) ([]model.TokenAccount, int64, error) {
	query := r.baseQuery(ctx, filter.IncludeDeleted)
	query = withTokenAccountFilters(query, filter)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := normalizeListLimit(filter.Limit)
	offset := normalizeListOffset(filter.Offset)

	var tokenAccounts []model.TokenAccount
	err := query.
		Order("id DESC").
		Limit(limit).
		Offset(offset).
		Find(&tokenAccounts).Error
	if err != nil {
		return nil, 0, err
	}

	return tokenAccounts, total, nil
}

func (r *TokenAccountRepo) Update(ctx context.Context, tokenAccount *model.TokenAccount) error {
	tx := r.baseQuery(ctx, false).
		Where("id = ?", tokenAccount.ID).
		Omit("id", "created_at", "deleted_at").
		Select("*").
		Updates(tokenAccount)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return errors.Join(ErrTokenAccountNotFound, v1.ErrNotFound)
	}
	return nil
}

func (r *TokenAccountRepo) Delete(ctx context.Context, id uint64) error {
	effected, err := gorm.G[model.TokenAccount](r.DB(ctx)).Where("id = ?", id).Delete(ctx)
	if err != nil {
		return err
	}
	if effected == 0 {
		return nil
	}
	return nil
}

func (r *TokenAccountRepo) Restore(ctx context.Context, id uint64) error {
	tx := r.DB(ctx).
		Model(&model.TokenAccount{}).
		Unscoped().
		Where("id = ?", id).
		Update("deleted_at", 0)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return ErrTokenAccountNotFound
	}
	return nil
}

func (r *TokenAccountRepo) baseQuery(ctx context.Context, includeDeleted bool) *gorm.DB {
	query := r.DB(ctx).Model(&model.TokenAccount{})
	//query := gorm.G[model.TokenAccount](r.DB(ctx)).Where("tenant_id = ?", r.TenantID)
	if includeDeleted {
		return query.Unscoped()
	}
	return query
}

func withTokenAccountFilters(query *gorm.DB, filter TokenAccountListFilter) *gorm.DB {
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.Email != "" {
		query = query.Where("email = ?", filter.Email)
	}
	if filter.AccountID != "" {
		query = query.Where("account_id = ?", filter.AccountID)
	}
	if filter.IDToken != "" {
		query = query.Where("id_token = ?", filter.IDToken)
	}
	return query
}

func normalizeListLimit(limit int) int {
	if limit <= 0 {
		return defaultTokenAccountListLimit
	}
	if limit > maxTokenAccountListLimit {
		return maxTokenAccountListLimit
	}
	return limit
}

func normalizeListOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}
