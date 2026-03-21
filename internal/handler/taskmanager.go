package handler

import (
	v1 "cpamgt/api/v1"
	"cpamgt/internal/server/task"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type TaskManageHandler struct {
	*Handler
	task.TaskManager
}

func NewTaskManageHandler(h *Handler, taskMgt task.TaskManager) *TaskManageHandler {
	return &TaskManageHandler{
		Handler:     h,
		TaskManager: taskMgt,
	}
}

func (h *TaskManageHandler) ListTasks(ctx *gin.Context) {
	states := h.ListTaskStates()
	resp := make([]v1.TaskStateResponse, 0, len(states))
	for _, state := range states {
		data := v1.TaskStateResponse{
			Name:           state.Name,
			Enabled:        state.Enabled,
			Interval:       int64(state.Interval.Milliseconds()),
			Timeout:        int64(state.Timeout.Milliseconds()),
			AllowOverlap:   state.AllowOverlap,
			Running:        state.Running,
			ActiveRuns:     state.ActiveRuns,
			LastStartedAt:  state.LastStartedAt,
			LastFinishedAt: state.LastFinishedAt,
			LastDuration:   int64(state.LastDuration.Milliseconds()),
			NextRunAt:      state.NextRunAt,
			LastError:      state.LastError,
			RunCount:       state.RunCount,
			SuccessCount:   state.SuccessCount,
			FailureCount:   state.FailureCount,
		}
		resp = append(resp, data)
	}
	v1.HandleSuccess(ctx, gin.H{
		"items": resp,
	})
}

func (h *TaskManageHandler) GetTask(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		v1.HandleError(c, v1.ErrBadRequest, nil)
		return
	}
	state, isExist := h.GetTaskState(name)
	if !isExist {
		v1.HandleError(c, v1.ErrNotFound, nil)
		return
	}
	v1.HandleSuccess(c, state)
}

func (h *TaskManageHandler) UpdateNamedTaskConfig(c *gin.Context) {
	taskName := strings.TrimSpace(c.Param("name"))
	if taskName == "" {
		v1.HandleError(c, v1.ErrBadRequest, nil)
		return
	}
	current, ok := h.GetTaskState(taskName)
	if !ok {
		v1.HandleError(c, v1.ErrNotFound, nil)
		return
	}
	var req v1.UpdateTaskConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		v1.HandleError(c, v1.ErrBadRequest, "invalid request body")
		return
	}
	if req.Enabled == nil && req.Interval == nil && req.Timeout == nil && req.AllowOverlap == nil {
		v1.HandleError(c, v1.ErrBadRequest, "at least one config field is required")
		return
	}
	cfg := task.TaskConfig{
		Enabled:      current.Enabled,
		Interval:     current.Interval,
		Timeout:      current.Timeout,
		AllowOverlap: current.AllowOverlap,
	}
	if req.Enabled != nil {
		cfg.Enabled = *req.Enabled
	}
	if req.AllowOverlap != nil {
		cfg.AllowOverlap = *req.AllowOverlap
	}
	if req.Interval != nil {
		interval, err := time.ParseDuration(strings.TrimSpace(*req.Interval))
		if err != nil {
			v1.HandleError(c, v1.ErrBadRequest, "invalid interval")
			return
		}
		cfg.Interval = interval
	}
	if req.Timeout != nil {
		timeout, err := time.ParseDuration(strings.TrimSpace(*req.Timeout))
		if err != nil {
			v1.HandleError(c, v1.ErrBadRequest, "invalid timeout")
			return
		}
		cfg.Timeout = timeout
	}
	if err := h.UpdateTaskConfig(taskName, cfg); err != nil {
		v1.HandleError(c, v1.ErrInternalServerError, err.Error())
		return
	}
	updated, ok := h.GetTaskState(taskName)
	if !ok {
		v1.HandleError(c, v1.ErrNotFound, "task not found")
		return
	}
	c.JSON(http.StatusOK, updated)
}
