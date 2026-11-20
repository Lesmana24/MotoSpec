package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/motospec/backend/internal/service"
)

type SpecHandler struct {
	specService *service.SpecService
}

func NewSpecHandler(specService *service.SpecService) *SpecHandler {
	return &SpecHandler{
		specService: specService,
	}
}

// SearchSpec handles GET /api/v1/specs/search?q=...
func (h *SpecHandler) SearchSpec(c *gin.Context) {
	query := c.Query("q")
	if len(query) < 2 {
		Error(c, http.StatusBadRequest, "INVALID_REQUEST", "Query parameter 'q' must be at least 2 characters.")
		return
	}

	result, err := h.specService.SearchAndSeed(c.Request.Context(), query)
	if err != nil {
		HandleDomainError(c, err)
		return
	}

	Success(c, http.StatusOK, result.Data, result.Meta)
}

// ListModels handles GET /api/v1/specs
func (h *SpecHandler) ListModels(c *gin.Context) {
	result, err := h.specService.ListSeededModels(c.Request.Context())
	if err != nil {
		HandleDomainError(c, err)
		return
	}

	Success(c, http.StatusOK, result.Data, result.Meta)
}
