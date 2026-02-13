package interfaces

import (
	"context"

	"github.com/akylbek/payment-system/fraud-service/internal/models"
)

// FraudRepository defines the contract for fraud data access
type FraudRepository interface {
	SaveDecision(ctx context.Context, paymentID, customerID string, amount float64, decision, reason string, riskScore int) error
	GetStats(ctx context.Context) (*models.FraudStats, error)
}
