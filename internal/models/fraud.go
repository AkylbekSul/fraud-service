package models

type FraudCheckRequest struct {
	PaymentID  string  `json:"payment_id"`
	Amount     float64 `json:"amount"`
	CustomerID string  `json:"customer_id"`
}

type FraudCheckResponse struct {
	Decision string `json:"decision"` // approve, deny, manual_review
	Reason   string `json:"reason"`
}

type FraudRule struct {
	ID          int
	Name        string
	MaxAmount   float64
	MaxPerHour  int
	Description string
}

type FraudStats struct {
	TotalChecks   int `json:"total_checks"`
	ApprovedCount int `json:"approved_count"`
	DeniedCount   int `json:"denied_count"`
	ManualReview  int `json:"manual_review_count"`
	AvgRiskScore  int `json:"avg_risk_score"`
}
