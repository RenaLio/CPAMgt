package codex

import (
	"encoding/json"
	"time"
)

type TokenData struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	AccountID    string    `json:"account_id"`
	IDToken      string    `json:"id_token"`
	Email        string    `json:"email"`
	Expire       time.Time `json:"expire"`
}

type UsageResponse struct {
	UserID               string          `json:"user_id"`
	AccountID            string          `json:"account_id"`
	Email                string          `json:"email"`
	PlanType             string          `json:"plan_type"`
	RateLimit            RateLimit       `json:"rate_limit"`
	CodeReviewRateLimit  RateLimit       `json:"code_review_rate_limit"`
	AdditionalRateLimits json.RawMessage `json:"additional_rate_limits"` // 可用 json.RawMessage 或 *struct{}
	Credits              Credits         `json:"credits"`
	Promo                Promo           `json:"promo"`
}

type RateLimit struct {
	Allowed         bool    `json:"allowed"`
	LimitReached    bool    `json:"limit_reached"`
	PrimaryWindow   Window  `json:"primary_window"`
	SecondaryWindow *Window `json:"secondary_window"`
}

type Window struct {
	UsedPercent        int   `json:"used_percent"`
	LimitWindowSeconds int64 `json:"limit_window_seconds"`
	ResetAfterSeconds  int64 `json:"reset_after_seconds"`
	ResetAt            int64 `json:"reset_at"`
}

type Credits struct {
	HasCredits          bool     `json:"has_credits"`
	Unlimited           bool     `json:"unlimited"`
	Balance             *float64 `json:"balance"`
	ApproxLocalMessages *int     `json:"approx_local_messages"`
	ApproxCloudMessages *int     `json:"approx_cloud_messages"`
}

type Promo struct {
	CampaignID string `json:"campaign_id"`
	Message    string `json:"message"`
}
