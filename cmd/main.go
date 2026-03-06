package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/akylbek/payment-system/fraud-service/internal/api"
	"github.com/akylbek/payment-system/fraud-service/internal/config"
	grpcserver "github.com/akylbek/payment-system/fraud-service/internal/grpcserver"
	"github.com/akylbek/payment-system/fraud-service/internal/repository"
	"github.com/akylbek/payment-system/fraud-service/internal/service"
	"github.com/akylbek/payment-system/fraud-service/internal/telemetry"
	fraudpb "github.com/akylbek/payment-system/proto/fraud"
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

	// Configure connection pool for high concurrency
	db.SetMaxOpenConns(30)
	db.SetMaxIdleConns(15)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Initialize repository
	fraudRepo := repository.NewFraudRepository(db)
	if err := fraudRepo.InitDB(); err != nil {
		telemetry.Logger.Fatal("Failed to initialize database", zap.Error(err))
	}

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisURL,
		PoolSize: 100,
	})

	// Initialize service
	fraudChecker := service.NewFraudChecker(redisClient)

	// Setup HTTP router (health, metrics, legacy HTTP endpoints)
	router := api.NewRouter(fraudRepo, fraudChecker)

	// Setup HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Start HTTP server in goroutine
	go func() {
		telemetry.Logger.Info("Fraud Service HTTP starting", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			telemetry.Logger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Start gRPC server
	grpcSrv := grpc.NewServer()
	fraudGRPCServer := grpcserver.NewFraudGRPCServer(fraudRepo, fraudChecker)
	fraudpb.RegisterFraudServiceServer(grpcSrv, fraudGRPCServer)

	grpcListener, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		telemetry.Logger.Fatal("Failed to listen for gRPC", zap.Error(err))
	}

	go func() {
		telemetry.Logger.Info("Fraud Service gRPC starting", zap.String("grpc_port", cfg.GRPCPort))
		if err := grpcSrv.Serve(grpcListener); err != nil {
			telemetry.Logger.Fatal("Failed to start gRPC server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	telemetry.Logger.Info("Shutting down server...")

	// Graceful shutdown gRPC
	grpcSrv.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		telemetry.Logger.Error("Server forced to shutdown", zap.Error(err))
	}

	telemetry.Logger.Info("Server exited")
}
