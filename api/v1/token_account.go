package v1

import (
	"cpamgt/internal/model"
	"encoding/json"
	"time"
)

type CreateTokenAccountRequest struct {
	IDToken      string          `json:"id_token" binding:"required"`
	AccessToken  string          `json:"access_token" binding:"required"`
	RefreshToken string          `json:"refresh_token"`
	AccountID    string          `json:"account_id" binding:"required"`
	Email        string          `json:"email" binding:"required"`
	Type         string          `json:"type" binding:"required"`
	Expired      *time.Time      `json:"expired,omitempty"`
	Extra        json.RawMessage `json:"extra,omitempty"`
}

type CreateTokenAccountBatchRequest struct {
	Items []CreateTokenAccountRequest `json:"items" binding:"required"`
}

type ListTokenAccountsRequest struct {
	Status   *model.TokenAccountStatus `form:"status,omitempty"`
	Page     int                       `form:"page,default=1" binding:"min=1"`
	PageSize int                       `form:"pageSize,default=20" binding:"min=1,max=200"`
}

type TokenAccountRespItem struct {
	ID               uint64          `json:"id"`
	TenantID         uint64          `json:"tenantId"`
	AccountID        string          `json:"accountId"`
	Email            string          `json:"email"`
	AccountType      string          `json:"type"`
	Status           string          `json:"status"`
	Percent          int64           `json:"percent"`
	CpaDelFlag       uint8           `json:"cpaFlag"`
	LastRefresh      *time.Time      `json:"lastRefresh"`
	QuotaRefreshTime *time.Time      `json:"quotaRefreshTime"`
	ExpiredAt        time.Time       `json:"expiredAt"`
	Extra            json.RawMessage `json:"extra"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
}

type ListTokenAccountsResponse = ListResponse[TokenAccountRespItem]
