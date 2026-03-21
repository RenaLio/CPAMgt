package cpa

type UsageResponse struct {
	FailedRequests int        `json:"failed_requests"`
	Usage          UsageStats `json:"usage"`
}

type UsageStats struct {
	TotalRequests  int                 `json:"total_requests"`
	SuccessCount   int                 `json:"success_count"`
	FailureCount   int                 `json:"failure_count"`
	TotalTokens    int                 `json:"total_tokens"`
	APIs           map[string]APIUsage `json:"apis"`
	RequestsByDay  map[string]int      `json:"requests_by_day"`
	RequestsByHour map[string]int      `json:"requests_by_hour"`
	TokensByDay    map[string]int      `json:"tokens_by_day"`
	TokensByHour   map[string]int      `json:"tokens_by_hour"`
}

type APIUsage struct {
	TotalRequests int                   `json:"total_requests"`
	TotalTokens   int                   `json:"total_tokens"`
	Models        map[string]ModelUsage `json:"models"`
}

type ModelUsage struct {
	TotalRequests int           `json:"total_requests"`
	TotalTokens   int           `json:"total_tokens"`
	Details       []UsageDetail `json:"details"`
}

type UsageDetail struct {
	Timestamp string     `json:"timestamp"`
	Source    string     `json:"source"`
	AuthIndex string     `json:"auth_index"`
	Tokens    TokenUsage `json:"tokens"`
	Failed    bool       `json:"failed"`
}

type TokenUsage struct {
	InputTokens     int `json:"input_tokens"`
	OutputTokens    int `json:"output_tokens"`
	ReasoningTokens int `json:"reasoning_tokens"`
	CachedTokens    int `json:"cached_tokens"`
	TotalTokens     int `json:"total_tokens"`
}

type AuthFilesResponse struct {
	Files []AuthFile `json:"files"`
}

type AuthFile struct {
	Account        string                 `json:"account"`
	AccountType    string                 `json:"account_type"`
	AuthIndex      string                 `json:"auth_index"`
	CreatedAt      string                 `json:"created_at"`
	Disabled       bool                   `json:"disabled"`
	Email          string                 `json:"email"`
	ID             string                 `json:"id"`
	IDToken        map[string]any         `json:"id_token"`
	Label          string                 `json:"label"`
	LastRefresh    string                 `json:"last_refresh"`
	ModTime        string                 `json:"modtime"`
	Name           string                 `json:"name"`
	NextRetryAfter string                 `json:"next_retry_after"`
	Path           string                 `json:"path"`
	Provider       string                 `json:"provider"`
	RuntimeOnly    bool                   `json:"runtime_only"`
	Size           int64                  `json:"size"`
	Source         string                 `json:"source"`
	Status         string                 `json:"status"`
	StatusMessage  string                 `json:"status_message"`
	Type           string                 `json:"type"`
	Unavailable    bool                   `json:"unavailable"`
	UpdatedAt      string                 `json:"updated_at"`
	Extra          map[string]interface{} `json:"-"`
}

type AuthFileStatusRequest struct {
	Name     string `json:"name"`
	Disabled bool   `json:"disabled"`
}
