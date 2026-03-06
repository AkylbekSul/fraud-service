package grpcserver

import (
	"context"

	"go.uber.org/zap"

	"github.com/akylbek/payment-system/fraud-service/internal/interfaces"
	"github.com/akylbek/payment-system/fraud-service/internal/models"
	"github.com/akylbek/payment-system/fraud-service/internal/service"
	"github.com/akylbek/payment-system/fraud-service/internal/telemetry"
	fraudpb "github.com/akylbek/payment-system/proto/fraud"
)

type FraudGRPCServer struct {
	fraudpb.UnimplementedFraudServiceServer
	repo    interfaces.FraudRepository
	checker *service.FraudChecker
}

func NewFraudGRPCServer(repo interfaces.FraudRepository, checker *service.FraudChecker) *FraudGRPCServer {
	return &FraudGRPCServer{
		repo:    repo,
		checker: checker,
	}
}

func (s *FraudGRPCServer) CheckFraud(ctx context.Context, req *fraudpb.CheckFraudRequest) (*fraudpb.CheckFraudResponse, error) {
	telemetry.Logger.Info("gRPC fraud check request",
		zap.String("payment_id", req.PaymentId),
		zap.Float64("amount", req.Amount),
		zap.String("customer_id", req.CustomerId),
	)

	fraudReq := &models.FraudCheckRequest{
		PaymentID:  req.PaymentId,
		Amount:     req.Amount,
		CustomerID: req.CustomerId,
	}

	decision := s.checker.CheckFraud(ctx, fraudReq)

	// Save decision to database
	riskScore := s.checker.CalculateRiskScore(fraudReq)
	if err := s.repo.SaveDecision(ctx, req.PaymentId, req.CustomerId, req.Amount, decision.Decision, decision.Reason, riskScore); err != nil {
		telemetry.Logger.Error("Error saving fraud decision",
			zap.String("payment_id", req.PaymentId),
			zap.Error(err),
		)
	}

	telemetry.Logger.Info("gRPC fraud check completed",
		zap.String("payment_id", req.PaymentId),
		zap.String("decision", decision.Decision),
		zap.String("reason", decision.Reason),
	)

	return &fraudpb.CheckFraudResponse{
		Decision: decision.Decision,
		Reason:   decision.Reason,
	}, nil
}
