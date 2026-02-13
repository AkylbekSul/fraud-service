package service

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/akylbek/payment-system/fraud-service/internal/models"
)

type FraudChecker struct {
	redisClient *redis.Client
}

func NewFraudChecker(redisClient *redis.Client) *FraudChecker {
	return &FraudChecker{redisClient: redisClient}
}

func (f *FraudChecker) CheckFraud(ctx context.Context, req *models.FraudCheckRequest) *models.FraudCheckResponse {
	// Rule 1: High amount check
	if req.Amount > 10000 {
		return &models.FraudCheckResponse{
			Decision: "deny",
			Reason:   "Amount exceeds $10,000 limit",
		}
	}

	// Rule 2: Velocity check (max 5 payments per hour)
	velocityKey := "fraud:velocity:" + req.CustomerID
	count, err := f.redisClient.Incr(ctx, velocityKey).Result()
	if err == nil {
		if count == 1 {
			f.redisClient.Expire(ctx, velocityKey, time.Hour)
		}
		if count > 5 {
			return &models.FraudCheckResponse{
				Decision: "deny",
				Reason:   "Too many payments in the last hour (velocity check failed)",
			}
		}
	}

	// Rule 3: High-value manual review
	if req.Amount > 5000 {
		return &models.FraudCheckResponse{
			Decision: "manual_review",
			Reason:   "High-value transaction requires manual review",
		}
	}

	return &models.FraudCheckResponse{
		Decision: "approve",
		Reason:   "All fraud checks passed",
	}
}

func (f *FraudChecker) CalculateRiskScore(req *models.FraudCheckRequest) int {
	score := 0

	if req.Amount > 1000 {
		score += 30
	}
	if req.Amount > 5000 {
		score += 50
	}

	return score
}
