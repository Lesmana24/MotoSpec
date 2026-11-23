package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/motospec/backend/internal/handler"
	"github.com/motospec/backend/internal/llm"
	"github.com/motospec/backend/internal/repository"
	"github.com/motospec/backend/internal/service"
	"github.com/motospec/backend/internal/worker"
)

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func loadEnvFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			if os.Getenv(k) == "" {
				os.Setenv(k, v)
			}
		}
	}
}

func main() {
	// Attempt to load .env from current directory or backend directory
	loadEnvFile(".env")
	loadEnvFile("backend/.env")
	loadEnvFile("../backend/.env")

	// Configure structured JSON logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("initializing MotoSpec backend service...")

	// Database Configuration (Default host port 5433 per docker-compose.yml)
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5433")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "motospec")
	dbSSLMode := getEnv("DB_SSLMODE", "disable")

	dsn := getEnv("DATABASE_URL", "")
	if dsn == "" {
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", dbUser, dbPassword, dbHost, dbPort, dbName, dbSSLMode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pgPool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		slog.Error("failed to create postgres connection pool", "error", err)
		os.Exit(1)
	}
	defer pgPool.Close()

	if err := pgPool.Ping(ctx); err != nil {
		slog.Warn("postgres ping failed during startup (will retry on requests)", "error", err)
	} else {
		slog.Info("connected to postgres database", "host", dbHost, "port", dbPort, "database", dbName)
	}

	// Redis Configuration
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       0,
	})
	defer redisClient.Close()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		slog.Warn("redis ping failed during startup (will retry on requests)", "error", err)
	} else {
		slog.Info("connected to redis cache", "addr", redisAddr)
	}

	// Repository Layer
	store := repository.NewStore(pgPool)
	specCache := repository.NewRedisSpecCache(redisClient, repository.DefaultSpecTTL)
	telemetryStore := repository.NewTelemetryStore()

	// LLM Orchestrator Layer
	geminiKey := getEnv("GEMINI_API_KEY", "")
	groqKey := getEnv("GROQ_API_KEY", "")
	groqModel := getEnv("GROQ_MODEL", "openai/gpt-oss-120b")

	primaryProvider := llm.NewPrimaryProvider(geminiKey, "", nil)
	fallbackProvider := llm.NewFallbackProvider(groqKey, "", groqModel, nil)
	llmOrchestrator := llm.NewOrchestrator(primaryProvider, fallbackProvider, telemetryStore)

	// Domain Service Layer
	specService := service.NewSpecService(store, specCache, llmOrchestrator, telemetryStore)
	garageService := service.NewGarageService(store)

	// Background Cron Worker
	odometerWorker := worker.NewOdometerWorker(store)
	if err := odometerWorker.Start(); err != nil {
		slog.Error("failed to start odometer cron worker", "error", err)
	}
	defer odometerWorker.Stop()

	// HTTP Handlers & Router Setup
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(handler.CORSMiddleware())
	router.Use(handler.UserIDMiddleware())
	router.Use(handler.OutageSimulationMiddleware())

	specHandler := handler.NewSpecHandler(specService)
	garageHandler := handler.NewGarageHandler(garageService)
	systemHandler := handler.NewSystemHandler(store, redisClient, telemetryStore, llmOrchestrator)

	// Route Registration matching docs/API_CONTRACT.md
	router.GET("/healthz", systemHandler.HealthCheck)

	apiV1 := router.Group("/api/v1")
	{
		// Spec Endpoints
		apiV1.GET("/specs/search", specHandler.SearchSpec)
		apiV1.GET("/specs", specHandler.ListModels)

		// Garage Endpoints
		garage := apiV1.Group("/garage")
		{
			garage.GET("/bikes", garageHandler.ListBikes)
			garage.POST("/bikes", garageHandler.AddBike)
			garage.GET("/bikes/:id", garageHandler.GetBikeDashboard)
			garage.PATCH("/bikes/:id/odometer", garageHandler.UpdateOdometer)
			garage.POST("/bikes/:id/maintenance", garageHandler.LogMaintenance)
			garage.GET("/bikes/:id/maintenance", garageHandler.ListMaintenanceHistory)
		}

		// System & Telemetry Endpoints
		apiV1.GET("/system/telemetry", systemHandler.GetTelemetry)
	}

	// Server Execution & Graceful Shutdown
	port := getEnv("PORT", "8080")
	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		slog.Info("starting HTTP API server", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server exited cleanly")
}
