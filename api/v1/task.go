package v1

import "time"

type UpdateTaskConfigRequest struct {
	Enabled      *bool   `json:"enabled,omitempty"`
	Interval     *string `json:"interval,omitempty"`
	Timeout      *string `json:"timeout,omitempty"`
	AllowOverlap *bool   `json:"allowOverlap,omitempty"`
}

type TaskStateResponse struct {
	Name           string     `json:"name"`
	Enabled        bool       `json:"enabled"`
	Interval       int64      `json:"interval"`
	Timeout        int64      `json:"timeout"`
	AllowOverlap   bool       `json:"allowOverlap"`
	Running        bool       `json:"running"`
	ActiveRuns     int        `json:"activeRuns"`
	LastStartedAt  *time.Time `json:"lastStartedAt,omitempty"`
	LastFinishedAt *time.Time `json:"lastFinishedAt,omitempty"`
	LastDuration   int64      `json:"lastDuration"`
	NextRunAt      *time.Time `json:"nextRunAt,omitempty"`
	LastError      string     `json:"lastError,omitempty"`
	RunCount       uint64     `json:"runCount"`
	SuccessCount   uint64     `json:"successCount"`
	FailureCount   uint64     `json:"failureCount"`
}
