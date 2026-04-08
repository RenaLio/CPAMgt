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
	data, err := h.tokenAccountService.List(c.Request.Context(), params)
	if err != nil {
		v1.HandleError(c, err, err.Error())
		return
	}
	resp := v1.ListResponse[v1.TokenAccountRespItem]{
		Page:     data.Page,
		PageSize: data.PageSize,
		Total:    data.Total,
		Items:    make([]v1.TokenAccountRespItem, len(data.Items)),
	}
	for i, item := range data.Items {
		resp.Items[i] = v1.TokenAccountRespItem{
			ID:               item.ID,
			TenantID:         item.TenantID,
			AccountID:        item.AccountID,
			Email:            item.Email,
			AccountType:      item.AccountType,
			Status:           string(item.Status),
			Percent:          item.Percent,
			CpaDelFlag:       item.CpaDelFlag,
			LastRefresh:      item.LastRefresh,
			QuotaRefreshTime: item.QuotaRefreshTime,
			ExpiredAt:        item.ExpiredAt,
			Extra:            item.Extra,
			CreatedAt:        item.CreatedAt,
			UpdatedAt:        item.UpdatedAt,
		}
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

func (h *TokenAccountHandler) RefreshTokenAccountQuota(c *gin.Context) {
	if err := h.tokenAccountService.ResetQuotaRefreshTime(c.Request.Context()); err != nil {
		v1.HandleError(c, err, err.Error())
		return
	}
	v1.HandleSuccess(c, nil)
}
