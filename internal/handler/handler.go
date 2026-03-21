package handler

import (
	"cpamgt/pkg/log"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	logger *log.Logger
}

func NewHandler(
	logger *log.Logger,
) *Handler {
	return &Handler{
		logger: logger,
	}
}

func (h *Handler) Log(ctx *gin.Context) *log.Logger {
	return h.logger.FromContext(ctx.Request.Context())
}
