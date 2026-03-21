package model

import (
	"encoding/json"
	"time"

	"gorm.io/plugin/soft_delete"
)

type TokenAccountStatus string

const (
	TokenAccountStatusAvailable      TokenAccountStatus = "available"
	TokenAccountStatusQuotaExhausted TokenAccountStatus = "quota_exhausted"
	TokenAccountStatusAuthExpired    TokenAccountStatus = "auth_expired"
)

func (s TokenAccountStatus) IsValid() bool {
	switch s {
	case TokenAccountStatusAvailable, TokenAccountStatusQuotaExhausted, TokenAccountStatusAuthExpired:
		return true
	default:
		return false
	}
}

type TokenAccount struct {
	ID               uint64                `gorm:"primaryKey" json:"id"`
	TenantID         uint64                `gorm:"column:tenant_id;not null;index" json:"tenantId"`
	IDToken          string                `gorm:"column:id_token;type:varchar(512);not null;uniqueIndex:idx_token_accounts_id_token" json:"idToken"`
	AccessToken      string                `gorm:"column:access_token;type:varchar(512);not null" json:"accessToken"`
	RefreshToken     string                `gorm:"column:refresh_token;type:varchar(512);not null" json:"refreshToken"`
	AccountID        string                `gorm:"column:account_id;type:varchar(128);not null;index" json:"accountId"`
	LastRefresh      *time.Time            `gorm:"column:last_refresh" json:"lastRefresh,omitempty"`
	Email            string                `gorm:"column:email;type:varchar(320);not null;index" json:"email"`
	AccountType      string                `gorm:"column:type;type:varchar(64);not null" json:"type"`
	ExpiredAt        time.Time             `gorm:"column:expired;not null;" json:"expired,omitempty"`
	Status           TokenAccountStatus    `gorm:"column:status;type:varchar(32);not null;default:'available';index" json:"status"`
	Percent          int64                 `gorm:"column:percent;not null;default:0;check:ck_token_accounts_percent,percent >= 0 AND percent <= 100" json:"percent"`
	QuotaRefreshTime *time.Time            `gorm:"column:quota_refresh_time" json:"quotaRefreshTime,omitempty"`
	CpaDelFlag       uint8                 `gorm:"column:cpa_flag;not null;default:1;comment:表示是否还要执行cpa删除操作，0-不执行，1-执行" json:"cpaFlag"`
	Extra            json.RawMessage       `gorm:"column:extra;type:json" json:"extra,omitempty"`
	CreatedAt        time.Time             `gorm:"column:created_at;not null;autoCreateTime" json:"createdAt"`
	UpdatedAt        time.Time             `gorm:"column:updated_at;not null;autoUpdateTime" json:"updatedAt"`
	DeletedAt        soft_delete.DeletedAt `gorm:"column:deleted_at;softDelete:milli;index;uniqueIndex:idx_token_accounts_id_token" json:"-"`
}

func (TokenAccount) TableName() string {
	return "token_accounts"
}
