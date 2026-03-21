package handler

import (
	v1 "cpamgt/api/v1"
	"cpamgt/internal/service"

	"github.com/gin-gonic/gin"
)

type CpaAccountHandler struct {
	*Handler
	svc service.CpaAccountService
}

func NewCpaAccountHandler(
	handler *Handler,
	svc service.CpaAccountService,
) *CpaAccountHandler {
	return &CpaAccountHandler{
		Handler: handler,
		svc:     svc,
	}
}

func (h *CpaAccountHandler) GetCpaConfig(c *gin.Context) {
	config, err := h.svc.GetCpaConfig(c.Request.Context())
	if err != nil {
		v1.HandleError(c, err, nil)
		return
	}
	v1.HandleSuccess(c, config)
}

func (h *CpaAccountHandler) SetCpaConfig(c *gin.Context) {
	var config v1.CpaAccountServiceConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		v1.HandleError(c, err, nil)
		return
	}
	if err := h.svc.SetCpaConfig(c.Request.Context(), &config); err != nil {
		v1.HandleError(c, err, nil)
		return
	}
	v1.HandleSuccess(c, nil)
}
