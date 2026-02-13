package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/akylbek/payment-system/fraud-service/internal/interfaces"
	"github.com/akylbek/payment-system/fraud-service/internal/models"
	"github.com/akylbek/payment-system/fraud-service/internal/service"
	"github.com/akylbek/payment-system/fraud-service/internal/telemetry"
)

type FraudHandler struct {
	repo    interfaces.FraudRepository
	checker *service.FraudChecker
}

func NewFraudHandler(repo interfaces.FraudRepository, checker *service.FraudChecker) *FraudHandler {
	return &FraudHandler{
		repo:    repo,
		checker: checker,
	}
}

func (h *FraudHandler) HandleFraudCheckRequest(msg *nats.Msg) {
	var req models.FraudCheckRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		telemetry.Logger.Error("Error unmarshaling fraud check request", zap.Error(err))
		return
	}

	telemetry.Logger.Info("Fraud check request",
		zap.String("payment_id", req.PaymentID),
		zap.Float64("amount", req.Amount),
		zap.String("customer_id", req.CustomerID),
	)

	ctx := context.Background()
	decision := h.checker.CheckFraud(ctx, &req)

	// Save decision to database
	riskScore := h.checker.CalculateRiskScore(&req)
	if err := h.repo.SaveDecision(ctx, req.PaymentID, req.CustomerID, req.Amount, decision.Decision, decision.Reason, riskScore); err != nil {
		telemetry.Logger.Error("Error saving fraud decision",
			zap.String("payment_id", req.PaymentID),
			zap.Error(err),
		)
	}

	// Send response back via NATS
	respJSON, _ := json.Marshal(decision)
	msg.Respond(respJSON)

	telemetry.Logger.Info("Fraud check completed",
		zap.String("payment_id", req.PaymentID),
		zap.String("decision", decision.Decision),
		zap.String("reason", decision.Reason),
	)
}

func (h *FraudHandler) GetFraudStats(c *gin.Context) {
	stats, err := h.repo.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch fraud stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}
