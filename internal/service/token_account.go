package service

import (
	"context"
	v1 "cpamgt/api/v1"
	"cpamgt/internal/external/auth/codex"
	"cpamgt/internal/model"
	"cpamgt/internal/repository"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type TokenAccountService interface {
	CreateTokenAccount(ctx context.Context, params *v1.CreateTokenAccountRequest) error
	CreateTokenAccounts(ctx context.Context, params *v1.CreateTokenAccountBatchRequest) int64
	GetTokenAccount(ctx context.Context, id int64) (*model.TokenAccount, error)
	List(ctx context.Context, params *v1.ListTokenAccountsRequest) (*v1.ListResponse[model.TokenAccount], error)
	ListAll(ctx context.Context) ([]model.TokenAccount, error)
	Update(ctx context.Context, input *UpdateTokenAccountInput) (*model.TokenAccount, error)
	Delete(ctx context.Context, id uint64) error
}

type tokenAccountService struct {
	*Service
	tokenRepo repository.TokenAccountRepository
}

func NewTokenAccountService(service *Service, tokenRepo repository.TokenAccountRepository) TokenAccountService {
	return &tokenAccountService{
		Service:   service,
		tokenRepo: tokenRepo,
	}
}

func (s *tokenAccountService) CreateTokenAccount(ctx context.Context, params *v1.CreateTokenAccountRequest) error {
	if params.Expired == nil {
		claims, err := codex.ParseJWTToken(params.AccessToken)
		if err != nil {
			return err
		}
		date := time.Unix(int64(claims.Exp), 0)
		params.Expired = &date
	}
	model := &model.TokenAccount{
		IDToken:      params.IDToken,
		AccessToken:  params.AccessToken,
		RefreshToken: params.RefreshToken,
		AccountID:    params.AccountID,
		Email:        params.Email,
		AccountType:  params.Type,
		ExpiredAt:    *params.Expired,
		Extra:        params.Extra,
	}
	return s.tokenRepo.Create(ctx, model)
}

func (s *tokenAccountService) CreateTokenAccounts(ctx context.Context, params *v1.CreateTokenAccountBatchRequest) int64 {
	models := make([]*model.TokenAccount, 0, len(params.Items))
	for _, item := range params.Items {
		if item.Expired == nil {
			claims, err := codex.ParseJWTToken(item.AccessToken)
			if err != nil {
				continue
			}
			date := time.Unix(int64(claims.Exp), 0)
			item.Expired = &date
		}
		model := &model.TokenAccount{
			IDToken:      item.IDToken,
			AccessToken:  item.AccessToken,
			RefreshToken: item.RefreshToken,
			AccountID:    item.AccountID,
			Email:        item.Email,
			AccountType:  item.Type,
			ExpiredAt:    *item.Expired,
			Extra:        item.Extra,
		}
		models = append(models, model)
	}

	return s.tokenRepo.CreateByBatch(ctx, models)
}

func (s *tokenAccountService) GetTokenAccount(ctx context.Context, id int64) (*model.TokenAccount, error) {
	return s.tokenRepo.GetByID(ctx, uint64(id), false)
}

func (s *tokenAccountService) List(ctx context.Context, params *v1.ListTokenAccountsRequest) (*v1.ListResponse[model.TokenAccount], error) {
	var err error
	resp := new(v1.ListResponse[model.TokenAccount])
	resp.Page = int64(params.Page)
	resp.PageSize = int64(params.PageSize)

	offset := (params.Page - 1) * params.PageSize
	wantTotal := params.PageSize
	curTotal := 0
	accounts := make([]model.TokenAccount, 0, params.PageSize)
	for {
		if curTotal >= wantTotal {
			break
		}
		page, total, err := s.tokenRepo.List(ctx, repository.TokenAccountListFilter{
			Status: params.Status,
			Limit:  wantTotal - curTotal,
			Offset: offset,
		})
		resp.Total = int64(total)
		if err != nil {
			return resp, err
		}
		curTotal += len(page)
		accounts = append(accounts, page...)
		offset += len(page)
		if int64(offset) >= total || len(page) == 0 {
			break
		}
	}
	resp.Items = accounts
	return resp, err
}

func (s *tokenAccountService) ListAll(ctx context.Context) ([]model.TokenAccount, error) {
	const pageSize = 200

	var accounts []model.TokenAccount
	offset := 0
	for {
		page, total, err := s.tokenRepo.List(ctx, repository.TokenAccountListFilter{
			Limit:  pageSize,
			Offset: offset,
		})
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, page...)
		offset += len(page)
		if int64(offset) >= total || len(page) == 0 {
			break
		}
	}
	return accounts, nil
}

type UpdateTokenAccountInput struct {
	ID               uint64
	IDToken          *string
	AccessToken      *string
	RefreshToken     *string
	AccountID        *string
	Email            *string
	AccountType      *string
	ExpiredAt        *time.Time
	Status           *model.TokenAccountStatus
	Percent          *int64
	QuotaRefreshTime *time.Time
	Extra            *json.RawMessage
	CpaDelFlag       *uint8
}

func (s *tokenAccountService) Update(ctx context.Context, input *UpdateTokenAccountInput) (*model.TokenAccount, error) {

	account, err := s.tokenRepo.GetByID(ctx, input.ID, false)
	if err != nil {
		return nil, err
	}

	if input.IDToken != nil {
		if strings.TrimSpace(*input.IDToken) == "" {
			return nil, errors.New("id_token is required")
		}
		account.IDToken = strings.TrimSpace(*input.IDToken)
	}
	if input.AccessToken != nil {
		if strings.TrimSpace(*input.AccessToken) == "" {
			return nil, errors.New("access_token is required")
		}
		account.AccessToken = strings.TrimSpace(*input.AccessToken)
	}
	if input.RefreshToken != nil {
		if strings.TrimSpace(*input.RefreshToken) == "" {
			return nil, errors.New("refresh_token is required")
		}
		account.RefreshToken = strings.TrimSpace(*input.RefreshToken)
	}
	if input.AccountID != nil {
		if strings.TrimSpace(*input.AccountID) == "" {
			return nil, errors.New("account_id is required")
		}
		account.AccountID = strings.TrimSpace(*input.AccountID)
	}
	if input.Email != nil {
		if strings.TrimSpace(*input.Email) == "" {
			return nil, errors.New("email is required")
		}
		account.Email = strings.TrimSpace(*input.Email)
	}
	if input.AccountType != nil {
		if strings.TrimSpace(*input.AccountType) == "" {
			return nil, errors.New("account_type is required")
		}
		account.AccountType = strings.TrimSpace(*input.AccountType)
	}
	if input.CpaDelFlag != nil {
		account.CpaDelFlag = *input.CpaDelFlag
	}
		
	if input.ExpiredAt != nil {
		account.ExpiredAt = *input.ExpiredAt
	}
	if input.Status != nil {
		if !input.Status.IsValid() {
			return nil, errors.New("status is invalid")
		}
		account.Status = *input.Status
	}
	if input.Percent != nil {
		if *input.Percent < 0 || *input.Percent > 100 {
			return nil, errors.New("percent is invalid")
		}
		account.Percent = int64(*input.Percent)
	}
	if input.QuotaRefreshTime != nil {
		account.QuotaRefreshTime = input.QuotaRefreshTime
	}
	if input.Extra != nil {
		if !json.Valid(*input.Extra) {
			return nil, errors.New("extra is not valid JSON")
		}
		account.Extra = normalizeExtra(*input.Extra)
	}

	if err := s.tokenRepo.Update(ctx, account); err != nil {
		return nil, err
	}
	return account, nil
}

func (s *tokenAccountService) Delete(ctx context.Context, id uint64) error {
	return s.tokenRepo.Delete(ctx, id)
}

func normalizeExtra(extra json.RawMessage) json.RawMessage {
	if len(extra) == 0 {
		return nil
	}
	trimmed := strings.TrimSpace(string(extra))
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	return json.RawMessage(trimmed)
}
