package model

import (
	"encoding/json"
	"time"

	"gorm.io/plugin/soft_delete"
)

type CpaAccount struct {
	ID        uint64                `gorm:"column:id;primaryKey" json:"id"`
	TenantID  uint64                `gorm:"column:tenant_id;not null;index" json:"tenantId"`
	BaseUrl   string                `gorm:"column:base_url;type:varchar(512);not null" json:"baseUrl"`
	Token     string                `gorm:"column:token;type:varchar(512);not null" json:"token"`
	Enable    int8                  `gorm:"column:enable;not null;default:1" json:"enable"`
	Extra     json.RawMessage       `gorm:"column:extra;type:json" json:"extra,omitempty"`
	CreatedAt time.Time             `gorm:"column:created_at;not null;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time             `gorm:"column:updated_at;not null;autoUpdateTime" json:"updatedAt"`
	DeletedAt soft_delete.DeletedAt `gorm:"column:deleted_at;softDelete:milli;index;uniqueIndex:idx_cpa_accounts_url" json:"-"`
}
