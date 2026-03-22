package handler

import (
	v1 "cpamgt/api/v1"
	"cpamgt/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TokenAccountHandler struct {
	*Handler
	tokenAccountService service.TokenAccountService
}

func NewTokenAccountHandler(
	handler *Handler,
	tokenAccountService service.TokenAccountService,
) *TokenAccountHandler {
	return &TokenAccountHandler{
		Handler:             handler,
		tokenAccountService: tokenAccountService,
	}
}

func (h *TokenAccountHandler) CreateTokenAccount(ctx *gin.Context) {
	params := new(v1.CreateTokenAccountRequest)
	if err := ctx.ShouldBindJSON(params); err != nil {
		v1.HandleError(ctx, v1.ErrBadRequest, err.Error())
		return
	}
	if err := h.tokenAccountService.CreateTokenAccount(ctx, params); err != nil {
		v1.HandleError(ctx, err, err.Error())
		return
	}
	v1.HandleSuccess(ctx, nil)
}

// todo: resp(total,success,datails[errorDetails])
func (h *TokenAccountHandler) CreateTokenAccountBatch(c *gin.Context) {
	params := new(v1.CreateTokenAccountBatchRequest)
	if err := c.ShouldBindJSON(params); err != nil {
		v1.HandleError(c, v1.ErrBadRequest, err.Error())
		return
	}
	successCount := h.tokenAccountService.CreateTokenAccounts(c.Request.Context(), params)

	v1.HandleSuccess(c, gin.H{
		"total":   len(params.Items),
		"success": successCount,
	})
}

func (h *TokenAccountHandler) GetTokenAccountByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		v1.HandleError(c, v1.ErrBadRequest, "empty id")
		return
	}
	intId, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		v1.HandleError(c, v1.ErrBadRequest, err.Error())
		return
	}
	_ = intId
	data, err := h.tokenAccountService.GetTokenAccount(c.Request.Context(), intId)
	if err != nil {
		v1.HandleError(c, err, err.Error())
		return
	}
	v1.HandleSuccess(c, data)
}

func (h *TokenAccountHandler) ListTokenAccounts(c *gin.Context) {
	params := new(v1.ListTokenAccountsRequest)
	if err := c.ShouldBind(params); err != nil {
		v1.HandleError(c, v1.ErrBadRequest, err.Error())
		return
	}
	resp, err := h.tokenAccountService.List(c.Request.Context(), params)
	if err != nil {
		v1.HandleError(c, err, err.Error())
		return
	}
	v1.HandleSuccess(c, resp)
}

func (h *TokenAccountHandler) HardDeleteBadAccount(c *gin.Context) {
	if err := h.tokenAccountService.HardDeleteBadAccount(c.Request.Context()); err != nil {
		v1.HandleError(c, err, err.Error())
		return
	}
	v1.HandleSuccess(c, nil)
}
