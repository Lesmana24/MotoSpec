package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/motospec/backend/internal/llm"
	"github.com/motospec/backend/internal/service"
)

type ErrorResponseEnvelope struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type SuccessResponseEnvelope struct {
	Meta any `json:"meta,omitempty"`
	Data any `json:"data"`
}

func Success(c *gin.Context, statusCode int, data any, meta ...any) {
	resp := SuccessResponseEnvelope{
		Data: data,
	}
	if len(meta) > 0 {
		resp.Meta = meta[0]
	}
	c.JSON(statusCode, resp)
}

func Error(c *gin.Context, statusCode int, code string, message string) {
	c.JSON(statusCode, ErrorResponseEnvelope{
		Error: ErrorBody{
			Code:    code,
			Message: message,
		},
	})
}

// HandleDomainError maps internal domain sentinel errors to standardized HTTP envelopes.
func HandleDomainError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		Error(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrInvalidOdometer):
		Error(c, http.StatusBadRequest, "INVALID_ODOMETER", err.Error())
	case errors.Is(err, service.ErrInvalidQuery):
		Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	case errors.Is(err, llm.ErrAllProvidersFailed):
		Error(c, http.StatusServiceUnavailable, "ALL_PROVIDERS_FAILED", "Both primary and fallback LLM providers failed or timed out.")
	default:
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected internal server error occurred.")
	}
}
