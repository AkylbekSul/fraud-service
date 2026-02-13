package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/akylbek/payment-system/fraud-service/internal/config"
	"github.com/akylbek/payment-system/fraud-service/internal/handlers"
	"github.com/akylbek/payment-system/fraud-service/internal/repository"
	"github.com/akylbek/payment-system/fraud-service/internal/service"
	"github.com/akylbek/payment-system/fraud-service/internal/telemetry"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize telemetry
	if err := telemetry.InitTelemetry("fraud-service"); err != nil {
		panic(fmt.Sprintf("Failed to initialize telemetry: %v", err))
	}
	defer telemetry.Shutdown(context.Background())

	telemetry.Logger.Info("Starting Fraud Service")

	// Connect to PostgreSQL
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		telemetry.Logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize repository
	fraudRepo := repository.NewFraudRepository(db)
	if err := fraudRepo.InitDB(); err != nil {
		telemetry.Logger.Fatal("Failed to initialize database", zap.Error(err))
	}

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})

	// Connect to NATS
	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		telemetry.Logger.Fatal("Failed to connect to NATS", zap.Error(err))
	}
	defer nc.Close()

	// Initialize service and handlers
	fraudChecker := service.NewFraudChecker(redisClient)
	fraudHandler := handlers.NewFraudHandler(fraudRepo, fraudChecker)

	// Subscribe to fraud check requests via NATS
	nc.Subscribe("fraud.check", fraudHandler.HandleFraudCheckRequest)
	telemetry.Logger.Info("Subscribed to fraud.check on NATS")

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(telemetry.TracingMiddleware())

	// Prometheus metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "fraud-service"})
	})

	r.GET("/fraud/stats", fraudHandler.GetFraudStats)

	// Setup HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// Start server in goroutine
	go func() {
		telemetry.Logger.Info("Fraud Service starting", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			telemetry.Logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	telemetry.Logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		telemetry.Logger.Error("Server forced to shutdown", zap.Error(err))
	}

	telemetry.Logger.Info("Server exited")
}
