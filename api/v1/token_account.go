package v1

import (
	"cpamgt/internal/model"
	"encoding/json"
	"time"
)

type CreateTokenAccountRequest struct {
	IDToken      string          `json:"id_token" binding:"required"`
	AccessToken  string          `json:"access_token" binding:"required"`
	RefreshToken string          `json:"refresh_token" binding:"required"`
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

type ListTokenAccountsResponse = ListResponse[model.TokenAccount]
