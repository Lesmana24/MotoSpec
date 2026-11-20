package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/motospec/backend/internal/repository/db"
	"github.com/motospec/backend/internal/service"
)

// mockStore implements repository.Store for unit tests
type mockStore struct {
	db.Querier
	execTxFn func(ctx context.Context, fn func(*db.Queries) error) error
	getUserBikeByIDFn func(ctx context.Context, arg db.GetUserBikeByIDParams) (db.GetUserBikeByIDRow, error)
	updateUserBikeOdometerFn func(ctx context.Context, arg db.UpdateUserBikeOdometerParams) (db.UserBike, error)
	getUserCompStatesFn func(ctx context.Context, userBikeID uuid.UUID) ([]db.GetUserComponentStatesWithSpecsRow, error)
}

func (m *mockStore) ExecTx(ctx context.Context, fn func(*db.Queries) error) error {
	if m.execTxFn != nil {
		return m.execTxFn(ctx, fn)
	}
	return nil
}

func (m *mockStore) Pool() *pgxpool.Pool {
	return nil
}

func (m *mockStore) GetUserBikeByID(ctx context.Context, arg db.GetUserBikeByIDParams) (db.GetUserBikeByIDRow, error) {
	if m.getUserBikeByIDFn != nil {
		return m.getUserBikeByIDFn(ctx, arg)
	}
	return db.GetUserBikeByIDRow{}, nil
}

func (m *mockStore) UpdateUserBikeOdometer(ctx context.Context, arg db.UpdateUserBikeOdometerParams) (db.UserBike, error) {
	if m.updateUserBikeOdometerFn != nil {
		return m.updateUserBikeOdometerFn(ctx, arg)
	}
	return db.UserBike{}, nil
}

func (m *mockStore) GetUserComponentStatesWithSpecs(ctx context.Context, userBikeID uuid.UUID) ([]db.GetUserComponentStatesWithSpecsRow, error) {
	if m.getUserCompStatesFn != nil {
		return m.getUserCompStatesFn(ctx, userBikeID)
	}
	return nil, nil
}

func TestCalculateComponentWear(t *testing.T) {
	svc := service.NewGarageService(nil)
	fixedNow := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	svc.SetNowFunc(func() time.Time { return fixedNow })

	km10000 := int32(10000)
	km16000 := int32(16000)
	date30DaysAgo := fixedNow.AddDate(0, 0, -30)
	date60DaysAgo := fixedNow.AddDate(0, 0, -60)
	date120DaysAgo := fixedNow.AddDate(0, 0, -120)

	tests := []struct {
		name              string
		currentKM         int32
		intervalKM        int32
		intervalMonths    int32
		lastKM            *int32
		lastAt            *time.Time
		expectedStatus    string
		expectedWearPct   *float64
		expectedNextDue   *int32
		expectedRemaining *int32
	}{
		{
			name:              "Cold Start - nil km and nil date -> unknown_needs_inspection",
			currentKM:         24000,
			intervalKM:        4000,
			intervalMonths:    4,
			lastKM:            nil,
			lastAt:            nil,
			expectedStatus:    service.StatusUnknownNeedsInspection,
			expectedWearPct:   nil,
			expectedNextDue:   nil,
			expectedRemaining: nil,
		},
		{
			name:              "Healthy Condition (< 75%)",
			currentKM:         18500,
			intervalKM:        4000,
			intervalMonths:    4,
			lastKM:            &km16000,       // 2500 km done out of 4000 -> 62.5%
			lastAt:            &date30DaysAgo, // 30 days out of 120 (4 months * 30) -> 25.0%
			expectedStatus:    service.StatusHealthy,
			expectedWearPct:   float64Ptr(62.5),
			expectedNextDue:   int32Ptr(20000),
			expectedRemaining: int32Ptr(1500),
		},
		{
			name:              "Warning Condition (75% <= Wear < 100%)",
			currentKM:         19200,
			intervalKM:        4000,
			intervalMonths:    4,
			lastKM:            &km16000,       // 3200 km done -> 80.0%
			lastAt:            &date60DaysAgo, // 60 days out of 120 -> 50.0%
			expectedStatus:    service.StatusWarning,
			expectedWearPct:   float64Ptr(80.0),
			expectedNextDue:   int32Ptr(20000),
			expectedRemaining: int32Ptr(800),
		},
		{
			name:              "Critical Condition - Distance Overdue (>= 100%)",
			currentKM:         20500,
			intervalKM:        4000,
			intervalMonths:    4,
			lastKM:            &km16000, // 4500 km done -> 112.5% uncapped
			lastAt:            &date60DaysAgo,
			expectedStatus:    service.StatusCritical,
			expectedWearPct:   float64Ptr(112.5),
			expectedNextDue:   int32Ptr(20000),
			expectedRemaining: int32Ptr(-500),
		},
		{
			name:              "Critical Condition - Time Trigger Dominates",
			currentKM:         11000,
			intervalKM:        4000,
			intervalMonths:    3,              // 90 days limit
			lastKM:            &km10000,       // 1000 km done -> 25.0%
			lastAt:            &date120DaysAgo, // 120 days out of 90 -> 133.3%
			expectedStatus:    service.StatusCritical,
			expectedWearPct:   float64Ptr(133.3),
			expectedNextDue:   int32Ptr(14000),
			expectedRemaining: int32Ptr(3000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wear, due, rem, status := svc.CalculateComponentWear(tt.currentKM, tt.intervalKM, tt.intervalMonths, tt.lastKM, tt.lastAt)
			if status != tt.expectedStatus {
				t.Errorf("status = %q, expected %q", status, tt.expectedStatus)
			}
			if !equalFloatPtr(wear, tt.expectedWearPct) {
				t.Errorf("wear = %v, expected %v", ptrVal(wear), ptrVal(tt.expectedWearPct))
			}
			if !equalInt32Ptr(due, tt.expectedNextDue) {
				t.Errorf("nextDue = %v, expected %v", ptrValInt(due), ptrValInt(tt.expectedNextDue))
			}
			if !equalInt32Ptr(rem, tt.expectedRemaining) {
				t.Errorf("remaining = %v, expected %v", ptrValInt(rem), ptrValInt(tt.expectedRemaining))
			}
		})
	}
}

func TestSortComponentsByUrgency(t *testing.T) {
	wear110 := 110.0
	wear85 := 85.0
	wear50 := 50.0

	items := []service.ComponentHealthItem{
		{Code: "spark_plug", Status: service.StatusHealthy, WearPercentage: &wear50, IntervalKM: 8000},
		{Code: "v_belt", Status: service.StatusUnknownNeedsInspection, WearPercentage: nil, IntervalKM: 24000},
		{Code: "engine_oil", Status: service.StatusCritical, WearPercentage: &wear110, IntervalKM: 4000},
		{Code: "brake_pads", Status: service.StatusWarning, WearPercentage: &wear85, IntervalKM: 10000},
	}

	service.SortComponentsByUrgency(items)

	expectedOrder := []string{"engine_oil", "brake_pads", "v_belt", "spark_plug"}
	for i, expected := range expectedOrder {
		if items[i].Code != expected {
			t.Errorf("index %d expected code %s, got %s", i, expected, items[i].Code)
		}
	}
}

func TestGarageService_UpdateOdometer_AntiRollback(t *testing.T) {
	bikeID := uuid.New()
	userID := "demo-user"

	store := &mockStore{
		getUserBikeByIDFn: func(ctx context.Context, arg db.GetUserBikeByIDParams) (db.GetUserBikeByIDRow, error) {
			return db.GetUserBikeByIDRow{
				ID:                bikeID,
				UserID:            userID,
				CurrentOdometerKm: 18500,
			}, nil
		},
	}

	svc := service.NewGarageService(store)

	// Attempt rollback: 17000 km < 18500 km
	_, err := svc.UpdateOdometer(context.Background(), userID, bikeID, service.UpdateOdometerRequest{
		CurrentOdometerKM: 17000,
		DailyEstimateKM:   30,
	})

	if err == nil {
		t.Fatalf("expected error on odometer rollback, got nil")
	}
	if !errors.Is(err, service.ErrInvalidOdometer) {
		t.Errorf("expected ErrInvalidOdometer, got: %v", err)
	}
}

func TestGarageService_UpdateOdometer_Success(t *testing.T) {
	bikeID := uuid.New()
	userID := "demo-user"

	currentKM := int32(18500)
	store := &mockStore{
		getUserBikeByIDFn: func(ctx context.Context, arg db.GetUserBikeByIDParams) (db.GetUserBikeByIDRow, error) {
			return db.GetUserBikeByIDRow{
				ID:                bikeID,
				UserID:            userID,
				CurrentOdometerKm: currentKM,
				Nickname:          "Vario 150",
				ModelDisplayName:  "Honda Vario 150",
			}, nil
		},
		updateUserBikeOdometerFn: func(ctx context.Context, arg db.UpdateUserBikeOdometerParams) (db.UserBike, error) {
			currentKM = arg.CurrentOdometerKm
			return db.UserBike{
				ID:                bikeID,
				CurrentOdometerKm: arg.CurrentOdometerKm,
			}, nil
		},
		getUserCompStatesFn: func(ctx context.Context, userBikeID uuid.UUID) ([]db.GetUserComponentStatesWithSpecsRow, error) {
			return nil, nil
		},
	}

	svc := service.NewGarageService(store)

	dashboard, err := svc.UpdateOdometer(context.Background(), userID, bikeID, service.UpdateOdometerRequest{
		CurrentOdometerKM: 19000,
		DailyEstimateKM:   35,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dashboard.CurrentOdometerKM != 19000 {
		t.Errorf("expected odometer 19000, got %d", dashboard.CurrentOdometerKM)
	}
}

// Helpers
func float64Ptr(v float64) *float64 { return &v }
func int32Ptr(v int32) *int32       { return &v }

func equalFloatPtr(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func equalInt32Ptr(a, b *int32) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func ptrVal(p *float64) string {
	if p == nil {
		return "nil"
	}
	return fmt.Sprintf("%.1f", *p)
}

func ptrValInt(p *int32) string {
	if p == nil {
		return "nil"
	}
	return fmt.Sprintf("%d", *p)
}
