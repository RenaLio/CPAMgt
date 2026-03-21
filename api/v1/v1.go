package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type ListResponse[T any] struct {
	Total    int64 `json:"total"`
	Items    []T   `json:"items"`
	Page     int64 `json:"page"`
	PageSize int64 `json:"pageSize"`
}

func HandleSuccess(ctx *gin.Context, data any) {
	if data == nil {
		data = map[string]any{}
	}
	resp := Response{Code: 0, Message: "success", Data: data}
	ctx.JSON(http.StatusOK, resp)
}

type AppError struct {
	Code     int
	HTTPCode int
	Message  string
}

func (e AppError) Error() string {
	return e.Message
}

func NewAppError(code int, httpCode int, message string) *AppError {
	return &AppError{
		Code:     code,
		HTTPCode: httpCode,
		Message:  message,
	}
}

func HandleError(ctx *gin.Context, err error, data interface{}) {
	if data == nil {
		data = map[string]any{}
	}
	if err == nil {
		HandleSuccess(ctx, data)
		return
	}
	if e, ok := errors.AsType[*AppError](err); ok {
		resp := Response{Code: e.Code, Message: e.Error(), Data: data}
		ctx.JSON(e.HTTPCode, resp)
		return
	}
	// log error for unknown error

	ctx.JSON(http.StatusInternalServerError, Response{Code: 500, Message: "unknown error", Data: data})
}
