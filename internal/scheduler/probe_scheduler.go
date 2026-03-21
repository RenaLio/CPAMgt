package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrTaskIsNil            = errors.New("task is nil")
	ErrTaskNameEmpty        = errors.New("task name is empty")
	ErrTaskAlreadyExists    = errors.New("task already exists")
	ErrTaskNotFound         = errors.New("task not found")
	ErrTaskIntervalInvalid  = errors.New("task interval must be greater than zero")
	ErrTaskTimeoutInvalid   = errors.New("task timeout must be greater than or equal to zero")
	ErrSchedulerTickInvalid = errors.New("scheduler tick must be greater than zero")
)

type Task interface {
	Name() string
	Run(ctx context.Context) error
}

type TaskConfig struct {
	Enabled      bool          `json:"enabled"`
	Interval     time.Duration `json:"interval"`
	Timeout      time.Duration `json:"timeout"`
	AllowOverlap bool          `json:"allowOverlap"`
}

type TaskState struct {
	Name           string        `json:"name"`
	Enabled        bool          `json:"enabled"`
	Interval       time.Duration `json:"interval"`
	Timeout        time.Duration `json:"timeout"`
	AllowOverlap   bool          `json:"allowOverlap"`
	Running        bool          `json:"running"`
	ActiveRuns     int           `json:"activeRuns"`
	LastStartedAt  *time.Time    `json:"lastStartedAt,omitempty"`
	LastFinishedAt *time.Time    `json:"lastFinishedAt,omitempty"`
	LastDuration   time.Duration `json:"lastDuration"`
	NextRunAt      *time.Time    `json:"nextRunAt,omitempty"`
	LastError      string        `json:"lastError,omitempty"`
	RunCount       uint64        `json:"runCount"`
	SuccessCount   uint64        `json:"successCount"`
	FailureCount   uint64        `json:"failureCount"`
}

type TaskServer struct {
	logger *slog.Logger
	tick   time.Duration

	mu    sync.RWMutex
	tasks map[string]*taskEntry

	runWG sync.WaitGroup
}

type taskEntry struct {
	task   Task
	config TaskConfig
	state  TaskState
}

type taskRunRequest struct {
	name      string
	task      Task
	timeout   time.Duration
	startedAt time.Time
}

func NewTaskServer(logger *slog.Logger, tick time.Duration) *TaskServer {
	if logger == nil {
		logger = slog.Default()
	}
	if tick <= 0 {
		tick = time.Second
	}

	return &TaskServer{
		logger: logger,
		tick:   tick,
		tasks:  make(map[string]*taskEntry),
	}
}

func (s *TaskServer) RegisterTask(task Task, cfg TaskConfig) error {
	if task == nil {
		return ErrTaskIsNil
	}
	if err := validateTaskConfig(cfg); err != nil {
		return err
	}

	name := strings.TrimSpace(task.Name())
	if name == "" {
		return ErrTaskNameEmpty
	}

	now := time.Now().UTC()
	entry := &taskEntry{
		task:   task,
		config: cfg,
		state: TaskState{
			Name:         name,
			Enabled:      cfg.Enabled,
			Interval:     cfg.Interval,
			Timeout:      cfg.Timeout,
			AllowOverlap: cfg.AllowOverlap,
		},
	}
	if cfg.Enabled {
		nextRunAt := now
		entry.state.NextRunAt = &nextRunAt
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[name]; exists {
		return ErrTaskAlreadyExists
	}
	s.tasks[name] = entry
	s.logger.Info(
		"task registered",
		"task", name,
		"enabled", cfg.Enabled,
		"interval", cfg.Interval.String(),
		"timeout", cfg.Timeout.String(),
		"allow_overlap", cfg.AllowOverlap,
	)
	return nil
}

func (s *TaskServer) UpdateTaskConfig(name string, cfg TaskConfig) error {
	if err := validateTaskConfig(cfg); err != nil {
		return err
	}
	taskName := strings.TrimSpace(name)
	if taskName == "" {
		return ErrTaskNameEmpty
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.tasks[taskName]
	if !ok {
		return ErrTaskNotFound
	}

	previous := entry.config
	entry.config = cfg
	entry.state.Enabled = cfg.Enabled
	entry.state.Interval = cfg.Interval
	entry.state.Timeout = cfg.Timeout
	entry.state.AllowOverlap = cfg.AllowOverlap
	if cfg.Enabled {
		if entry.state.NextRunAt == nil || previous.Interval != cfg.Interval {
			nextRunAt := time.Now().UTC().Add(cfg.Interval)
			entry.state.NextRunAt = &nextRunAt
		}
	} else {
		entry.state.NextRunAt = nil
	}
	return nil
}

func (s *TaskServer) SetTaskEnabled(name string, enabled bool) error {
	taskName := strings.TrimSpace(name)
	if taskName == "" {
		return ErrTaskNameEmpty
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.tasks[taskName]
	if !ok {
		return ErrTaskNotFound
	}

	entry.config.Enabled = enabled
	entry.state.Enabled = enabled
	if enabled {
		nextRunAt := time.Now().UTC()
		entry.state.NextRunAt = &nextRunAt
	} else {
		entry.state.NextRunAt = nil
	}
	return nil
}

func (s *TaskServer) SetTaskInterval(name string, interval time.Duration) error {
	if interval <= 0 {
		return ErrTaskIntervalInvalid
	}

	taskName := strings.TrimSpace(name)
	if taskName == "" {
		return ErrTaskNameEmpty
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.tasks[taskName]
	if !ok {
		return ErrTaskNotFound
	}

	entry.config.Interval = interval
	entry.state.Interval = interval
	if entry.state.NextRunAt != nil {
		nextRunAt := time.Now().UTC().Add(interval)
		entry.state.NextRunAt = &nextRunAt
	}
	return nil
}

func (s *TaskServer) ListTaskStates() []TaskState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	states := make([]TaskState, 0, len(s.tasks))
	for _, entry := range s.tasks {
		states = append(states, cloneTaskState(entry.state))
	}

	sort.Slice(states, func(i, j int) bool {
		return states[i].Name < states[j].Name
	})
	return states
}

func (s *TaskServer) GetTaskState(name string) (TaskState, bool) {
	taskName := strings.TrimSpace(name)
	if taskName == "" {
		return TaskState{}, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.tasks[taskName]
	if !ok {
		return TaskState{}, false
	}
	return cloneTaskState(entry.state), true
}

func (s *TaskServer) Start(ctx context.Context) error {
	if s.tick <= 0 {
		return ErrSchedulerTickInvalid
	}

	s.logger.Info("task server started", "tick", s.tick.String())
	ticker := time.NewTicker(s.tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("task server stopping", "reason", ctx.Err())
			waitCh := make(chan struct{})
			go func() {
				s.runWG.Wait()
				close(waitCh)
			}()

			select {
			case <-waitCh:
			case <-time.After(30 * time.Second):
				s.logger.Warn("task server stop timeout reached, there are still running tasks")
			}

			s.logger.Info("task server stopped")
			return nil
		case <-ticker.C:
			s.dispatchDueTasks(ctx, time.Now().UTC())
		}
	}
}

func (s *TaskServer) dispatchDueTasks(ctx context.Context, now time.Time) {
	requests := make([]taskRunRequest, 0, len(s.tasks))

	s.mu.Lock()
	for name, entry := range s.tasks {
		cfg := entry.config
		state := &entry.state

		if !cfg.Enabled {
			continue
		}
		if state.NextRunAt != nil && now.Before(*state.NextRunAt) {
			continue
		}
		if state.Running && !cfg.AllowOverlap {
			continue
		}

		startedAt := now
		nextRunAt := startedAt.Add(cfg.Interval)

		state.Running = true
		state.ActiveRuns++
		state.LastStartedAt = &startedAt
		state.NextRunAt = &nextRunAt
		state.RunCount++

		requests = append(requests, taskRunRequest{
			name:      name,
			task:      entry.task,
			timeout:   cfg.Timeout,
			startedAt: startedAt,
		})
	}
	s.mu.Unlock()

	for i := range requests {
		s.runWG.Add(1)
		go s.executeTask(ctx, requests[i])
	}
}

func (s *TaskServer) executeTask(ctx context.Context, request taskRunRequest) {
	defer s.runWG.Done()

	runCtx := ctx
	cancel := func() {}
	if request.timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, request.timeout)
	}
	defer cancel()

	err := request.task.Run(runCtx)
	finishedAt := time.Now().UTC()
	duration := finishedAt.Sub(request.startedAt)

	s.mu.Lock()
	entry, ok := s.tasks[request.name]
	if ok {
		state := &entry.state
		if state.ActiveRuns > 0 {
			state.ActiveRuns--
		}
		state.Running = state.ActiveRuns > 0
		state.LastFinishedAt = &finishedAt
		state.LastDuration = duration
		if err != nil {
			state.LastError = err.Error()
			state.FailureCount++
		} else {
			state.LastError = ""
			state.SuccessCount++
		}
	}
	s.mu.Unlock()

	if err != nil {
		s.logger.Error("task execution failed", "task", request.name, "duration", duration.String(), "err", err)
		return
	}
	s.logger.Info("task execution finished", "task", request.name, "duration", duration.String())
}

func validateTaskConfig(cfg TaskConfig) error {
	if cfg.Interval <= 0 {
		return ErrTaskIntervalInvalid
	}
	if cfg.Timeout < 0 {
		return ErrTaskTimeoutInvalid
	}
	return nil
}

func cloneTaskState(src TaskState) TaskState {
	dst := src
	dst.LastStartedAt = cloneTimePtr(src.LastStartedAt)
	dst.LastFinishedAt = cloneTimePtr(src.LastFinishedAt)
	dst.NextRunAt = cloneTimePtr(src.NextRunAt)
	return dst
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
