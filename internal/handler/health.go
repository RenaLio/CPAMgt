package handler

import (
	v1 "cpamgt/api/v1"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	*Handler
}

func NewHealthHandler(handler *Handler) *HealthHandler {
	return &HealthHandler{
		Handler: handler,
	}
}

func (h *HealthHandler) GetHealth(ctx *gin.Context) {
	v1.HandleSuccess(ctx, gin.H{"status": "ok"})
}

func (h *HealthHandler) Ping(ctx *gin.Context) {
	v1.HandleSuccess(ctx, "pong")
}
