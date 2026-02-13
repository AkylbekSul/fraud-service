package repository

import (
	"context"
	"database/sql"

	"github.com/akylbek/payment-system/fraud-service/internal/models"
)

type FraudRepository struct {
	db *sql.DB
}

func NewFraudRepository(db *sql.DB) *FraudRepository {
	return &FraudRepository{db: db}
}

func (r *FraudRepository) InitDB() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS fraud_rules (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			max_amount DECIMAL(15,2),
			max_per_hour INTEGER,
			description TEXT,
			active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS fraud_decisions (
			id SERIAL PRIMARY KEY,
			payment_id VARCHAR(255) NOT NULL,
			customer_id VARCHAR(255) NOT NULL,
			amount DECIMAL(15,2) NOT NULL,
			decision VARCHAR(50) NOT NULL,
			reason TEXT,
			risk_score INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_fraud_decisions_payment_id ON fraud_decisions(payment_id)`,
		`CREATE INDEX IF NOT EXISTS idx_fraud_decisions_customer_id ON fraud_decisions(customer_id)`,
	}

	for _, query := range queries {
		if _, err := r.db.Exec(query); err != nil {
			return err
		}
	}

	// Insert default rules
	r.db.Exec(`
		INSERT INTO fraud_rules (name, max_amount, max_per_hour, description)
		VALUES 
			('High Amount Check', 10000.00, NULL, 'Deny payments over $10,000'),
			('Velocity Check', NULL, 5, 'Max 5 payments per hour per customer')
		ON CONFLICT DO NOTHING
	`)

	return nil
}

func (r *FraudRepository) SaveDecision(ctx context.Context, paymentID, customerID string, amount float64, decision, reason string, riskScore int) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fraud_decisions (payment_id, customer_id, amount, decision, reason, risk_score)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, paymentID, customerID, amount, decision, reason, riskScore)
	return err
}

func (r *FraudRepository) GetStats(ctx context.Context) (*models.FraudStats, error) {
	var stats models.FraudStats
	err := r.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN decision = 'approve' THEN 1 END) as approved,
			COUNT(CASE WHEN decision = 'deny' THEN 1 END) as denied,
			COUNT(CASE WHEN decision = 'manual_review' THEN 1 END) as manual_review,
			COALESCE(AVG(risk_score), 0) as avg_risk_score
		FROM fraud_decisions
	`).Scan(&stats.TotalChecks, &stats.ApprovedCount, &stats.DeniedCount,
		&stats.ManualReview, &stats.AvgRiskScore)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}
