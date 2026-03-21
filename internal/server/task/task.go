package task

import (
	"context"
	"cpamgt/pkg/log"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	DefaultTick         = time.Second * 1
	DefaultEnabled      = false
	DefaultInterval     = time.Second * 30
	DefaultTimeoutTask  = time.Second * 0
	DefaultAllowOverlap = false
)

var (
	ErrTaskIsNil                = errors.New("task is nil")
	ErrTaskNameEmpty            = errors.New("task name is empty")
	ErrTaskAlreadyExists        = errors.New("task already exists")
	ErrTaskNotFound             = errors.New("task not found")
	ErrTaskIntervalInvalid      = errors.New("task interval must be greater than zero")
	ErrTaskTimeoutInvalid       = errors.New("task timeout must be greater than or equal to zero")
	ErrSchedulerTickInvalid     = errors.New("scheduler tick must be greater than zero")
	ErrTaskServerAlreadyRunning = errors.New("task server already running")
)

type TaskServer struct {
	logger *log.Logger
	tick   time.Duration
	mu     *sync.RWMutex
	tasks  map[string]*taskEntry
	runWG  *sync.WaitGroup

	lifecycleMu *sync.Mutex
	started     bool
	cancel      context.CancelFunc
	done        chan struct{}
}

func NewTaskServer(logger *log.Logger, tasks ...Task) *TaskServer {
	server := new(TaskServer)
	server.logger = logger
	server.tick = DefaultTick
	server.mu = new(sync.RWMutex)
	server.runWG = new(sync.WaitGroup)
	server.lifecycleMu = new(sync.Mutex)
	server.tasks = make(map[string]*taskEntry)
	for _, task := range tasks {
		defaultConfig := TaskConfig{
			Enabled:      DefaultEnabled,
			Interval:     DefaultInterval,
			Timeout:      DefaultTimeoutTask,
			AllowOverlap: DefaultAllowOverlap,
		}
		err := server.RegisterTask(task, defaultConfig)
		if err != nil {
			server.logger.Error("failed to register task", "error", err)
			panic(err)
		}
	}

	return server
}

type Task interface {
	Name() string
	Run(ctx context.Context) error
}

type TaskManager interface {
	ListTaskStates() []TaskState
	GetTaskState(name string) (TaskState, bool)
	UpdateTaskConfig(name string, cfg TaskConfig) error
}

func GetTaskManager(t *TaskServer) TaskManager {
	return t
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

func (t *TaskServer) Start(ctx context.Context) error {
	if t.tick <= 0 {
		return ErrSchedulerTickInvalid
	}
	if ctx == nil {
		ctx = context.Background()
	}

	runCtx, cancel := context.WithCancel(ctx)

	// make sure the task server is not already running
	t.lifecycleMu.Lock()

	if t.started {
		t.lifecycleMu.Unlock()
		cancel()
		return ErrTaskServerAlreadyRunning
	}

	t.started = true             // mark the task server as started
	t.cancel = cancel            // store the cancel function for later use
	t.done = make(chan struct{}) // create a channel to signal when the task server is done
	doneCh := t.done
	t.lifecycleMu.Unlock()

	defer func() {
		// func to close the channel and reset the task server state
		t.lifecycleMu.Lock()
		if t.done == doneCh {
			// signal the task server is done by closing the channel
			close(doneCh)
			t.done = nil
		}
		t.cancel = nil
		t.started = false
		t.lifecycleMu.Unlock()
	}()

	t.logger.Info("task server started", "tick", t.tick.String())
	ticker := time.NewTicker(t.tick)
	defer ticker.Stop()

	for {
		select {
		case <-runCtx.Done():
			t.logger.Info("task server stopping", "reason", ctx.Err())
			waitCh := make(chan struct{})
			go func() {
				t.runWG.Wait()
				close(waitCh)
			}()

			select {
			case <-waitCh:
			case <-time.After(10 * time.Second):
				t.logger.Warn("task server stop timeout reached, there are still running tasks")
			}

			t.logger.Info("task server stopped")
			return nil
		case <-ticker.C:
			t.dispatchDueTasks(ctx, time.Now().UTC())
		}
	}
}

func (t *TaskServer) Stop(ctx context.Context) error {
	var ctxCancel context.CancelFunc
	if ctx == nil {
		ctx, ctxCancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer ctxCancel()
	}

	t.lifecycleMu.Lock()
	if !t.started {
		t.lifecycleMu.Unlock()
		return nil
	}
	cancel := t.cancel
	doneCh := t.done
	t.lifecycleMu.Unlock()

	if cancel != nil {
		cancel()
	}
	if doneCh == nil {
		return nil
	}

	select {
	case <-doneCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *TaskServer) RegisterTask(task Task, cfg TaskConfig) error {
	entry, err := buildEntry(task, cfg)
	if err != nil {
		return err
	}
	name := strings.TrimSpace(entry.task.Name())
	if name == "" {
		return ErrTaskNameEmpty
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.tasks[name]; exists {
		return ErrTaskAlreadyExists
	}
	t.tasks[name] = entry
	t.logger.Info(
		"task registered",
		"task", name,
		"enabled", cfg.Enabled,
		"interval", cfg.Interval.String(),
		"timeout", cfg.Timeout.String(),
		"allow_overlap", cfg.AllowOverlap,
	)
	return nil
}

func (t *TaskServer) UpdateTaskConfig(name string, cfg TaskConfig) error {
	if err := validateTaskConfig(cfg); err != nil {
		return err
	}
	taskName := strings.TrimSpace(name)
	if taskName == "" {
		return ErrTaskNameEmpty
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	entry, ok := t.tasks[taskName]
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

func (t *TaskServer) SetTaskEnabled(name string, enabled bool) error {
	taskName := strings.TrimSpace(name)
	if taskName == "" {
		return ErrTaskNameEmpty
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	entry, ok := t.tasks[taskName]
	if !ok {
		return ErrTaskNotFound
	}

	entry.config.Enabled = enabled
	entry.state.Enabled = enabled
	if enabled {
		nextRunAt := time.Now().UTC().Add(entry.config.Interval)
		entry.state.NextRunAt = &nextRunAt
	} else {
		entry.state.NextRunAt = nil
	}
	return nil
}

func (t *TaskServer) SetTaskInterval(name string, interval time.Duration) error {
	if interval <= 0 {
		return ErrTaskIntervalInvalid
	}

	taskName := strings.TrimSpace(name)
	if taskName == "" {
		return ErrTaskNameEmpty
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	entry, ok := t.tasks[taskName]
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

func (t *TaskServer) ListTaskStates() []TaskState {
	t.mu.RLock()
	defer t.mu.RUnlock()

	states := make([]TaskState, 0, len(t.tasks))
	for _, entry := range t.tasks {
		states = append(states, cloneTaskState(entry.state))
	}

	sort.Slice(states, func(i, j int) bool {
		return states[i].Name < states[j].Name
	})
	return states
}

func (t *TaskServer) GetTaskState(name string) (TaskState, bool) {
	taskName := strings.TrimSpace(name)
	if taskName == "" {
		return TaskState{}, false
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	entry, ok := t.tasks[taskName]
	if !ok {
		return TaskState{}, false
	}
	return cloneTaskState(entry.state), true
}

func (t *TaskServer) dispatchDueTasks(ctx context.Context, now time.Time) {
	requests := make([]taskRunRequest, 0, len(t.tasks))

	t.mu.Lock()
	for name, entry := range t.tasks {
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
	t.mu.Unlock()

	for i := range requests {
		t.runWG.Add(1)
		go t.executeTask(ctx, requests[i])
	}
}

func (t *TaskServer) executeTask(ctx context.Context, request taskRunRequest) {
	defer t.runWG.Done()

	runCtx := ctx
	cancel := func() {}
	if request.timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, request.timeout)
	}
	defer cancel()

	err := request.task.Run(runCtx)
	finishedAt := time.Now().UTC()
	duration := finishedAt.Sub(request.startedAt)

	t.mu.Lock()
	entry, ok := t.tasks[request.name]
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
	t.mu.Unlock()

	if err != nil {
		t.logger.Error("task execution failed", "task", request.name, "duration", duration.String(), "err", err)
		return
	}
	t.logger.Info("task execution finished", "task", request.name, "duration", duration.String())
}

func buildEntry(task Task, cfg TaskConfig) (*taskEntry, error) {
	if task == nil {
		return nil, ErrTaskIsNil
	}
	if err := validateTaskConfig(cfg); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(task.Name())
	if name == "" {
		return nil, ErrTaskNameEmpty
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
	return entry, nil
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
