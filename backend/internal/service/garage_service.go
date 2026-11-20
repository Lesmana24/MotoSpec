package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/motospec/backend/internal/repository"
	"github.com/motospec/backend/internal/repository/db"
)

var (
	ErrInvalidOdometer = errors.New("service: new odometer cannot be lower than current odometer")
)

// Status constants
const (
	StatusUnknownNeedsInspection = "unknown_needs_inspection"
	StatusCritical               = "critical"
	StatusWarning                = "warning"
	StatusHealthy                = "healthy"
)

// Request and Response DTOs for Garage

type AddBikeRequest struct {
	BikeModelID          uuid.UUID                 `json:"bike_model_id" binding:"required"`
	Nickname             string                    `json:"nickname" binding:"required,min=1,max=100"`
	CurrentOdometerKM    int32                     `json:"current_odometer_km" binding:"gte=0"`
	DailyEstimateKM      int32                     `json:"daily_estimate_km" binding:"gte=0"`
	AutoIncrementEnabled bool                      `json:"auto_increment_enabled"`
	InitialBaselines     []ComponentBaselineInput `json:"initial_baselines"`
}

type ComponentBaselineInput struct {
	SpecComponentID uuid.UUID  `json:"spec_component_id" binding:"required"`
	LastReplacedKM  *int32     `json:"last_replaced_km,omitempty"`
	LastReplacedAt  *time.Time `json:"last_replaced_at,omitempty"`
}

type UpdateOdometerRequest struct {
	CurrentOdometerKM    int32 `json:"current_odometer_km" binding:"gte=0"`
	DailyEstimateKM      int32 `json:"daily_estimate_km" binding:"gte=0"`
	AutoIncrementEnabled bool  `json:"auto_increment_enabled"`
}

type MaintenanceLogItem struct {
	SpecComponentID uuid.UUID `json:"spec_component_id" binding:"required"`
	CostIDR         *int32    `json:"cost_idr,omitempty"`
	Notes           string    `json:"notes"`
}

type LogMaintenanceRequest struct {
	OdometerKM int32                `json:"odometer_km" binding:"gte=0"`
	ServicedAt time.Time            `json:"serviced_at" binding:"required"`
	Items      []MaintenanceLogItem `json:"items" binding:"required,min=1,dive"`
}

type HealthSummary struct {
	HealthyCount  int `json:"healthy_count"`
	WarningCount  int `json:"warning_count"`
	CriticalCount int `json:"critical_count"`
	UnknownCount  int `json:"unknown_count"`
}

type UserBikeSummaryItem struct {
	ID                   uuid.UUID     `json:"id"`
	Nickname             string        `json:"nickname"`
	ModelDisplayName     string        `json:"model_display_name"`
	Category             string        `json:"category"`
	CurrentOdometerKM    int32         `json:"current_odometer_km"`
	DailyEstimateKM      int32         `json:"daily_estimate_km"`
	AutoIncrementEnabled bool          `json:"auto_increment_enabled"`
	Summary              HealthSummary `json:"summary"`
}

type ComponentHealthItem struct {
	SpecComponentID uuid.UUID  `json:"spec_component_id"`
	Code            string     `json:"code"`
	Name            string     `json:"name"`
	Category        string     `json:"category"`
	IntervalKM      int32      `json:"interval_km"`
	IntervalMonths  int32      `json:"interval_months"`
	UrgencyLevel    string     `json:"urgency_level"`
	Notes           string     `json:"notes"`
	LastReplacedKM  *int32     `json:"last_replaced_km"`
	LastReplacedAt  *time.Time `json:"last_replaced_at"`
	NextDueKM       *int32     `json:"next_due_km"`
	RemainingKM     *int32     `json:"remaining_km"`
	WearPercentage  *float64   `json:"wear_percentage"`
	Status          string     `json:"status"`
}

type BikeHealthDashboard struct {
	ID                   uuid.UUID             `json:"id"`
	Nickname             string                `json:"nickname"`
	ModelDisplayName     string                `json:"model_display_name"`
	Category             string                `json:"category"`
	EngineCC             int32                 `json:"engine_cc"`
	CurrentOdometerKM    int32                 `json:"current_odometer_km"`
	DailyEstimateKM      int32                 `json:"daily_estimate_km"`
	AutoIncrementEnabled bool                  `json:"auto_increment_enabled"`
	LastOdometerSyncAt   time.Time             `json:"last_odometer_sync_at"`
	Summary              HealthSummary         `json:"summary"`
	Components           []ComponentHealthItem `json:"components"`
}

type GarageService struct {
	store repository.Store
	nowFn func() time.Time // Injectable for deterministic tests
}

func NewGarageService(store repository.Store) *GarageService {
	return &GarageService{
		store: store,
		nowFn: time.Now,
	}
}

// SetNowFunc overrides current time (for testing time-based wear calculations).
func (s *GarageService) SetNowFunc(fn func() time.Time) {
	s.nowFn = fn
}

// CalculateComponentWear implements the dual-trigger wear formula and status classification.
func (s *GarageService) CalculateComponentWear(
	currentKM int32,
	intervalKM int32,
	intervalMonths int32,
	lastKM *int32,
	lastAt *time.Time,
) (wearPct *float64, nextDue *int32, remaining *int32, status string) {
	now := s.nowFn()

	// 1. Cold start / No history
	if lastKM == nil && lastAt == nil {
		return nil, nil, nil, StatusUnknownNeedsInspection
	}

	var wearDistance *float64
	var wearTime *float64

	// Distance wear calculation
	if lastKM != nil && intervalKM > 0 {
		diffKM := float64(currentKM - *lastKM)
		kmRatio := (diffKM / float64(intervalKM)) * 100.0
		wearDistance = &kmRatio

		due := *lastKM + intervalKM
		rem := due - currentKM
		nextDue = &due
		remaining = &rem
	}

	// Time wear calculation
	if lastAt != nil && intervalMonths > 0 {
		daysSince := float64(now.Sub(*lastAt).Hours() / 24.0)
		if daysSince < 0 {
			daysSince = 0
		}
		expectedDays := float64(intervalMonths * 30)
		timeRatio := (daysSince / expectedDays) * 100.0
		wearTime = &timeRatio
	}

	// Dual-trigger evaluation
	var effectiveWear float64
	if wearDistance != nil && wearTime != nil {
		effectiveWear = math.Max(*wearDistance, *wearTime)
	} else if wearDistance != nil {
		effectiveWear = *wearDistance
	} else if wearTime != nil {
		effectiveWear = *wearTime
	}

	// Rule: Minimum 0%, not capped at 100%
	if effectiveWear < 0 {
		effectiveWear = 0.0
	}
	roundedWear := math.Round(effectiveWear*10.0) / 10.0
	wearPct = &roundedWear

	// Status classification
	switch {
	case roundedWear >= 100.0:
		status = StatusCritical
	case roundedWear >= 75.0:
		status = StatusWarning
	default:
		status = StatusHealthy
	}

	return wearPct, nextDue, remaining, status
}

// SortComponentsByUrgency sorts components according to priority:
// critical -> warning -> unknown_needs_inspection -> healthy.
func SortComponentsByUrgency(items []ComponentHealthItem) {
	statusPriority := map[string]int{
		StatusCritical:               1,
		StatusWarning:                2,
		StatusUnknownNeedsInspection: 3,
		StatusHealthy:                4,
	}

	sort.SliceStable(items, func(i, j int) bool {
		pi := statusPriority[items[i].Status]
		pj := statusPriority[items[j].Status]
		if pi != pj {
			return pi < pj
		}

		// Secondary sort: wear percentage descending if present
		if items[i].WearPercentage != nil && items[j].WearPercentage != nil {
			if *items[i].WearPercentage != *items[j].WearPercentage {
				return *items[i].WearPercentage > *items[j].WearPercentage
			}
		}

		// Fallback to lower intervalKM
		return items[i].IntervalKM < items[j].IntervalKM
	})
}

// ListUserBikes returns all motorcycles in the user's garage with component health summary counters.
func (s *GarageService) ListUserBikes(ctx context.Context, userID string) ([]UserBikeSummaryItem, error) {
	bikes, err := s.store.ListUserBikes(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user bikes: %w", err)
	}

	items := make([]UserBikeSummaryItem, len(bikes))
	for i, b := range bikes {
		// Fetch components to compute summary
		compStates, err := s.store.GetUserComponentStatesWithSpecs(ctx, b.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get component states for bike %s: %w", b.ID, err)
		}

		var summary HealthSummary
		for _, cs := range compStates {
			_, _, _, status := s.CalculateComponentWear(b.CurrentOdometerKm, cs.IntervalKm, cs.IntervalMonths, cs.LastReplacedKm, cs.LastReplacedAt)
			switch status {
			case StatusHealthy:
				summary.HealthyCount++
			case StatusWarning:
				summary.WarningCount++
			case StatusCritical:
				summary.CriticalCount++
			case StatusUnknownNeedsInspection:
				summary.UnknownCount++
			}
		}

		items[i] = UserBikeSummaryItem{
			ID:                   b.ID,
			Nickname:             b.Nickname,
			ModelDisplayName:     b.ModelDisplayName,
			Category:             b.ModelCategory,
			CurrentOdometerKM:    b.CurrentOdometerKm,
			DailyEstimateKM:      b.DailyEstimateKm,
			AutoIncrementEnabled: b.AutoIncrementEnabled,
			Summary:              summary,
		}
	}

	return items, nil
}

// AddUserBike registers a new motorcycle into the user garage and initializes component baselines.
func (s *GarageService) AddUserBike(ctx context.Context, userID string, req AddBikeRequest) (*BikeHealthDashboard, error) {
	model, err := s.store.GetBikeModelByID(ctx, req.BikeModelID)
	if err != nil {
		return nil, fmt.Errorf("%w: bike model not found", ErrNotFound)
	}

	comps, err := s.store.ListSpecComponentsByModelID(ctx, model.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list model components: %w", err)
	}

	// Map initial baselines by spec_component_id
	baselineMap := make(map[uuid.UUID]ComponentBaselineInput, len(req.InitialBaselines))
	for _, b := range req.InitialBaselines {
		baselineMap[b.SpecComponentID] = b
	}

	var createdBike db.UserBike
	err = s.store.ExecTx(ctx, func(q *db.Queries) error {
		bike, err := q.CreateUserBike(ctx, db.CreateUserBikeParams{
			UserID:               userID,
			BikeModelID:          model.ID,
			Nickname:             req.Nickname,
			CurrentOdometerKm:    req.CurrentOdometerKM,
			DailyEstimateKm:      req.DailyEstimateKM,
			AutoIncrementEnabled: req.AutoIncrementEnabled,
		})
		if err != nil {
			return fmt.Errorf("failed to create user bike: %w", err)
		}
		createdBike = bike

		// Insert component states (omitted baselines default to nil)
		for _, comp := range comps {
			var lastKM *int32
			var lastAt *time.Time

			if input, ok := baselineMap[comp.ID]; ok {
				lastKM = input.LastReplacedKM
				lastAt = input.LastReplacedAt
			}

			_, err := q.UpsertUserComponentState(ctx, db.UpsertUserComponentStateParams{
				UserBikeID:        bike.ID,
				SpecComponentID:   comp.ID,
				LastReplacedKm:    lastKM,
				LastReplacedAt:    lastAt,
			})
			if err != nil {
				return fmt.Errorf("failed to initialize component state: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.GetBikeDashboard(ctx, userID, createdBike.ID)
}

// GetBikeDashboard returns full wear metrics, health summary, and sorted component schedules.
func (s *GarageService) GetBikeDashboard(ctx context.Context, userID string, bikeID uuid.UUID) (*BikeHealthDashboard, error) {
	bike, err := s.store.GetUserBikeByID(ctx, db.GetUserBikeByIDParams{
		ID:     bikeID,
		UserID: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: bike not found", ErrNotFound)
	}

	states, err := s.store.GetUserComponentStatesWithSpecs(ctx, bike.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get component states: %w", err)
	}

	componentItems := make([]ComponentHealthItem, len(states))
	var summary HealthSummary

	for i, cs := range states {
		wearPct, nextDue, remaining, status := s.CalculateComponentWear(
			bike.CurrentOdometerKm,
			cs.IntervalKm,
			cs.IntervalMonths,
			cs.LastReplacedKm,
			cs.LastReplacedAt,
		)

		switch status {
		case StatusHealthy:
			summary.HealthyCount++
		case StatusWarning:
			summary.WarningCount++
		case StatusCritical:
			summary.CriticalCount++
		case StatusUnknownNeedsInspection:
			summary.UnknownCount++
		}

		componentItems[i] = ComponentHealthItem{
			SpecComponentID: cs.SpecComponentID,
			Code:            cs.Code,
			Name:            cs.Name,
			Category:        cs.Category,
			IntervalKM:      cs.IntervalKm,
			IntervalMonths:  cs.IntervalMonths,
			UrgencyLevel:    cs.UrgencyLevel,
			Notes:           cs.Notes,
			LastReplacedKM:  cs.LastReplacedKm,
			LastReplacedAt:  cs.LastReplacedAt,
			NextDueKM:       nextDue,
			RemainingKM:     remaining,
			WearPercentage:  wearPct,
			Status:          status,
		}
	}

	SortComponentsByUrgency(componentItems)

	return &BikeHealthDashboard{
		ID:                   bike.ID,
		Nickname:             bike.Nickname,
		ModelDisplayName:     bike.ModelDisplayName,
		Category:             bike.ModelCategory,
		EngineCC:             bike.ModelEngineCc,
		CurrentOdometerKM:    bike.CurrentOdometerKm,
		DailyEstimateKM:      bike.DailyEstimateKm,
		AutoIncrementEnabled: bike.AutoIncrementEnabled,
		LastOdometerSyncAt:   bike.LastOdometerSyncAt,
		Summary:              summary,
		Components:           componentItems,
	}, nil
}

// UpdateOdometer validates monotonicity and updates mileage and auto-increment settings.
func (s *GarageService) UpdateOdometer(ctx context.Context, userID string, bikeID uuid.UUID, req UpdateOdometerRequest) (*BikeHealthDashboard, error) {
	current, err := s.store.GetUserBikeByID(ctx, db.GetUserBikeByIDParams{
		ID:     bikeID,
		UserID: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: bike not found", ErrNotFound)
	}

	if req.CurrentOdometerKM < current.CurrentOdometerKm {
		return nil, fmt.Errorf("%w: new odometer (%d km) cannot be lower than current (%d km)",
			ErrInvalidOdometer, req.CurrentOdometerKM, current.CurrentOdometerKm)
	}

	_, err = s.store.UpdateUserBikeOdometer(ctx, db.UpdateUserBikeOdometerParams{
		ID:                   bikeID,
		UserID:               userID,
		CurrentOdometerKm:    req.CurrentOdometerKM,
		DailyEstimateKm:      req.DailyEstimateKM,
		AutoIncrementEnabled: req.AutoIncrementEnabled,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update bike odometer: %w", err)
	}

	return s.GetBikeDashboard(ctx, userID, bikeID)
}

// LogMaintenance logs batch component maintenance, resets baselines, and bumps odometer if higher.
func (s *GarageService) LogMaintenance(ctx context.Context, userID string, bikeID uuid.UUID, req LogMaintenanceRequest) (*BikeHealthDashboard, error) {
	current, err := s.store.GetUserBikeByID(ctx, db.GetUserBikeByIDParams{
		ID:     bikeID,
		UserID: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: bike not found", ErrNotFound)
	}

	err = s.store.ExecTx(ctx, func(q *db.Queries) error {
		// 1. If maintenance odometer > current bike odometer, atomically bump bike odometer
		if req.OdometerKM > current.CurrentOdometerKm {
			_, err := q.BumpUserBikeOdometerIfHigher(ctx, db.BumpUserBikeOdometerIfHigherParams{
				ID:                  bikeID,
				CurrentOdometerKm: req.OdometerKM,
			})
			if err != nil {
				return fmt.Errorf("failed to bump odometer: %w", err)
			}
		}

		// 2. Insert maintenance logs and reset baseline for each serviced component
		for _, item := range req.Items {
			_, err := q.CreateMaintenanceLog(ctx, db.CreateMaintenanceLogParams{
				UserBikeID:      bikeID,
				SpecComponentID: item.SpecComponentID,
				OdometerKm:      req.OdometerKM,
				ServicedAt:      req.ServicedAt,
				CostIdr:         item.CostIDR,
				Notes:           item.Notes,
			})
			if err != nil {
				return fmt.Errorf("failed to create maintenance log: %w", err)
			}

			servicedAtCopy := req.ServicedAt
			odometerKmCopy := req.OdometerKM
			_, err = q.ResetComponentStateOnMaintenance(ctx, db.ResetComponentStateOnMaintenanceParams{
				UserBikeID:      bikeID,
				SpecComponentID: item.SpecComponentID,
				LastReplacedKm:  &odometerKmCopy,
				LastReplacedAt:  &servicedAtCopy,
			})
			if err != nil {
				return fmt.Errorf("failed to reset component state: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.GetBikeDashboard(ctx, userID, bikeID)
}

// ListMaintenanceHistory returns the chronological maintenance log records for a motorcycle.
func (s *GarageService) ListMaintenanceHistory(ctx context.Context, userID string, bikeID uuid.UUID) ([]db.ListMaintenanceLogsByBikeRow, error) {
	// Verify bike exists and belongs to user
	_, err := s.store.GetUserBikeByID(ctx, db.GetUserBikeByIDParams{
		ID:     bikeID,
		UserID: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: bike not found", ErrNotFound)
	}

	logs, err := s.store.ListMaintenanceLogsByBike(ctx, bikeID)
	if err != nil {
		return nil, fmt.Errorf("failed to list maintenance logs: %w", err)
	}

	return logs, nil
}
