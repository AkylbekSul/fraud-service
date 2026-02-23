package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/akylbek/payment-system/fraud-service/internal/handlers"
	"github.com/akylbek/payment-system/fraud-service/internal/interfaces"
	"github.com/akylbek/payment-system/fraud-service/internal/service"
	"github.com/akylbek/payment-system/fraud-service/internal/telemetry"
)

func NewRouter(repo interfaces.FraudRepository, checker *service.FraudChecker) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(telemetry.TracingMiddleware())

	// Prometheus metrics
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "fraud-service"})
	})

	// Fraud handlers
	fraudHandler := handlers.NewFraudHandler(repo, checker)
	r.POST("/fraud/check", fraudHandler.HandleFraudCheck)
	r.GET("/fraud/stats", fraudHandler.GetFraudStats)

	return r
}
