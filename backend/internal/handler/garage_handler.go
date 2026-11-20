package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/motospec/backend/internal/service"
)

type GarageHandler struct {
	garageService *service.GarageService
}

func NewGarageHandler(garageService *service.GarageService) *GarageHandler {
	return &GarageHandler{
		garageService: garageService,
	}
}

// ListBikes handles GET /api/v1/garage/bikes
func (h *GarageHandler) ListBikes(c *gin.Context) {
	userID := GetUserID(c)
	startTime := time.Now()

	bikes, err := h.garageService.ListUserBikes(c.Request.Context(), userID)
	if err != nil {
		HandleDomainError(c, err)
		return
	}

	meta := gin.H{
		"source":     "postgres_db",
		"latency_ms": time.Since(startTime).Milliseconds(),
	}
	Success(c, http.StatusOK, bikes, meta)
}

// AddBike handles POST /api/v1/garage/bikes
func (h *GarageHandler) AddBike(c *gin.Context) {
	userID := GetUserID(c)
	var req service.AddBikeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	dashboard, err := h.garageService.AddUserBike(c.Request.Context(), userID, req)
	if err != nil {
		HandleDomainError(c, err)
		return
	}

	meta := gin.H{
		"source":     "postgres_db",
		"latency_ms": 12,
	}
	Success(c, http.StatusCreated, dashboard, meta)
}

// GetBikeDashboard handles GET /api/v1/garage/bikes/:id
func (h *GarageHandler) GetBikeDashboard(c *gin.Context) {
	userID := GetUserID(c)
	bikeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		Error(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid motorcycle UUID format")
		return
	}

	startTime := time.Now()
	dashboard, err := h.garageService.GetBikeDashboard(c.Request.Context(), userID, bikeID)
	if err != nil {
		HandleDomainError(c, err)
		return
	}

	meta := gin.H{
		"source":     "postgres_db",
		"latency_ms": time.Since(startTime).Milliseconds(),
	}
	Success(c, http.StatusOK, dashboard, meta)
}

// UpdateOdometer handles PATCH /api/v1/garage/bikes/:id/odometer
func (h *GarageHandler) UpdateOdometer(c *gin.Context) {
	userID := GetUserID(c)
	bikeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		Error(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid motorcycle UUID format")
		return
	}

	var req service.UpdateOdometerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	startTime := time.Now()
	dashboard, err := h.garageService.UpdateOdometer(c.Request.Context(), userID, bikeID, req)
	if err != nil {
		HandleDomainError(c, err)
		return
	}

	meta := gin.H{
		"source":     "postgres_db",
		"latency_ms": time.Since(startTime).Milliseconds(),
	}
	Success(c, http.StatusOK, dashboard, meta)
}

// LogMaintenance handles POST /api/v1/garage/bikes/:id/maintenance
func (h *GarageHandler) LogMaintenance(c *gin.Context) {
	userID := GetUserID(c)
	bikeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		Error(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid motorcycle UUID format")
		return
	}

	var req service.LogMaintenanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	startTime := time.Now()
	dashboard, err := h.garageService.LogMaintenance(c.Request.Context(), userID, bikeID, req)
	if err != nil {
		HandleDomainError(c, err)
		return
	}

	meta := gin.H{
		"source":     "postgres_db",
		"latency_ms": time.Since(startTime).Milliseconds(),
	}
	Success(c, http.StatusCreated, dashboard, meta)
}

// ListMaintenanceHistory handles GET /api/v1/garage/bikes/:id/maintenance
func (h *GarageHandler) ListMaintenanceHistory(c *gin.Context) {
	userID := GetUserID(c)
	bikeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		Error(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid motorcycle UUID format")
		return
	}

	startTime := time.Now()
	history, err := h.garageService.ListMaintenanceHistory(c.Request.Context(), userID, bikeID)
	if err != nil {
		HandleDomainError(c, err)
		return
	}

	meta := gin.H{
		"source":     "postgres_db",
		"latency_ms": time.Since(startTime).Milliseconds(),
	}
	Success(c, http.StatusOK, history, meta)
}
