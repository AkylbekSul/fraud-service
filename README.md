# Fraud Service

The Fraud Service is a microservice responsible for real-time fraud detection and risk assessment for payment transactions. It uses rule-based checks and behavioral analysis to identify potentially fraudulent activities.

## Overview

The Fraud Service provides:
- Real-time fraud detection for payment transactions
- Risk score calculation based on multiple factors
- Velocity checks using Redis for rate limiting
- Fraud statistics and analytics
- Asynchronous communication via NATS
- Historical fraud data storage in PostgreSQL

## Architecture

```
Payment Orchestrator → NATS (fraud.check) → Fraud Service
                                                  ↓
                                           Risk Analysis
                                                  ↓
                                      Redis (velocity checks)
                                                  ↓
                                      PostgreSQL (fraud logs)
                                                  ↓
                                    NATS Response (approved/rejected)
```

## Technology Stack

- **Language**: Go 1.21+
- **Web Framework**: Gin
- **Database**: PostgreSQL
- **Cache**: Redis (for velocity checks)
- **Message Broker**: NATS (for request/response)
- **Observability**: OpenTelemetry + Jaeger + Prometheus
- **Logging**: Zap (Uber)

## Configuration

The service uses environment variables for configuration:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8083` |
| `DATABASE_URL` | PostgreSQL connection string | Required |
| `REDIS_URL` | Redis server address | Required |
| `NATS_URL` | NATS server address | Required |
| `JAEGER_ENDPOINT` | Jaeger collector endpoint | Optional |

## API Endpoints

### Get Fraud Statistics
```http
GET /fraud/stats
```

**Response**: `200 OK`
```json
{
  "total_checks": 1500,
  "fraud_detected": 45,
  "fraud_rate": 0.03,
  "last_updated": "2026-02-13T10:00:00Z"
}
```

### Health Check
```http
GET /health
```

**Response**: `200 OK`
```json
{
  "status": "ok",
  "service": "fraud-service"
}
```

### Prometheus Metrics
```http
GET /metrics
```

## NATS Communication

### Subscribe to: `fraud.check`

The service listens for fraud check requests on the `fraud.check` subject.

**Request Message**:
```json
{
  "payment_id": "payment-uuid",
  "customer_id": "customer-123",
  "amount": 100.50,
  "currency": "USD",
  "merchant_id": "merchant-456"
}
```

**Response Message**:
```json
{
  "payment_id": "payment-uuid",
  "approved": true,
  "risk_score": 0.15,
  "reason": "Low risk transaction",
  "checked_at": "2026-02-13T10:00:00Z"
}
```

## Fraud Detection Rules

The service implements multiple fraud detection mechanisms:

### 1. Amount-Based Checks
- **High Amount Threshold**: Transactions over $10,000 are flagged
- **Unusual Amount Patterns**: Amounts just below reporting thresholds

### 2. Velocity Checks (Redis-based)
- **Transaction Frequency**: Maximum transactions per customer per time window
- **Spending Velocity**: Total amount spent in rolling time windows
- **Geographic Velocity**: Multiple transactions from different locations

Example velocity rules:
```go
// Maximum 10 transactions per hour per customer
// Maximum $5,000 spending per hour per customer
// Maximum 3 different merchants per 10 minutes
```

### 3. Behavioral Analysis
- **First-time Customer**: Higher scrutiny for new accounts
- **Merchant Risk Profile**: Known high-risk merchant categories
- **Time-based Patterns**: Unusual transaction times

### 4. Blacklist Checks
- Customer blacklist
- Merchant blacklist
- IP address blacklist (if available)

## Risk Score Calculation

Risk scores range from 0.0 (no risk) to 1.0 (high risk):

```
Risk Score = Σ (Rule Weight × Rule Result)
```

Example weights:
- High amount: 0.3
- High velocity: 0.4
- Blacklisted: 1.0
- First-time customer: 0.1
- High-risk merchant: 0.2

**Thresholds**:
- `score < 0.3`: Approved
- `0.3 ≤ score < 0.7`: Review recommended
- `score ≥ 0.7`: Rejected

## Key Features

### Real-time Processing
- Sub-100ms average response time
- Asynchronous via NATS for non-blocking operation
- Redis-backed velocity checks for performance

### Fraud Logging
All fraud checks are logged to PostgreSQL:
```sql
CREATE TABLE fraud_checks (
    id SERIAL PRIMARY KEY,
    payment_id VARCHAR(255) NOT NULL,
    customer_id VARCHAR(255) NOT NULL,
    risk_score DECIMAL(5,4) NOT NULL,
    approved BOOLEAN NOT NULL,
    reason TEXT,
    checked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### Observability
- Prometheus metrics for fraud detection rates
- Distributed tracing with OpenTelemetry
- Structured logging with correlation IDs
- Alert on high fraud rates

## Running the Service

### Local Development

1. Set up environment variables:
```bash
export DATABASE_URL="postgresql://user:password@localhost:5432/fraud_service?sslmode=disable"
export REDIS_URL="localhost:6379"
export NATS_URL="nats://localhost:4222"
export PORT="8083"
```

2. Run database migrations:
```bash
# Apply migrations from migrations/ directory
```

3. Start the service:
```bash
go run cmd/main.go
```

### Docker

Build and run with Docker:
```bash
docker build -t fraud-service .
docker run -p 8083:8083 \
  -e DATABASE_URL="..." \
  -e REDIS_URL="..." \
  -e NATS_URL="..." \
  fraud-service
```

### Docker Compose

Run as part of the complete system:
```bash
docker-compose up fraud-service
```

## Dependencies

### External Services
- **PostgreSQL**: Fraud check history and analytics
- **Redis**: Velocity tracking and rate limiting
- **NATS**: Request/response messaging with orchestrator
- **Jaeger** (optional): Distributed tracing

### Integration Points
- **Payment Orchestrator**: Main consumer of fraud checks
- **Analytics Service**: Can consume fraud statistics

## Monitoring

### Prometheus Metrics

Custom metrics exposed:
```
# Fraud check metrics
fraud_checks_total{result="approved|rejected"}
fraud_check_duration_seconds
fraud_risk_score_histogram

# Velocity metrics
velocity_checks_total
velocity_limit_exceeded_total

# System metrics
http_request_duration_seconds
http_requests_total
```

### Key Alerts

Recommended alerts:
```yaml
# High fraud rate
- alert: HighFraudRate
  expr: rate(fraud_checks_total{result="rejected"}[5m]) > 0.1
  
# Slow fraud checks
- alert: SlowFraudChecks
  expr: histogram_quantile(0.95, fraud_check_duration_seconds) > 0.1

# Service unavailable
- alert: FraudServiceDown
  expr: up{job="fraud-service"} == 0
```

## Development

### Project Structure
```
fraud-service/
├── cmd/
│   └── main.go                 # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go           # Configuration management
│   ├── handlers/
│   │   └── fraud_handler.go    # NATS and HTTP handlers
│   ├── interfaces/
│   │   └── fraud_repository.go # Repository interface
│   ├── models/
│   │   └── fraud.go            # Domain models
│   ├── repository/
│   │   └── fraud_repository.go # Data access layer
│   ├── service/
│   │   └── fraud_checker.go    # Fraud detection logic
│   └── telemetry/
│       └── telemetry.go        # Observability setup
├── migrations/
│   └── 001_fraud_service_schema.sql
├── Dockerfile
└── README.md
```

### Running Tests
```bash
go test ./...
```

### Adding New Fraud Rules

To add a new fraud detection rule:

1. Implement the check in `internal/service/fraud_checker.go`
2. Add appropriate weight to risk score calculation
3. Update unit tests
4. Update documentation

Example:
```go
func (fc *FraudChecker) CheckNewRule(ctx context.Context, payment *models.Payment) (bool, float64) {
    // Implement rule logic
    if ruleViolated {
        return false, 0.25 // weight
    }
    return true, 0.0
}
```

## Performance Considerations

### Redis Connection Pooling
The service uses Redis connection pooling for optimal performance:
```go
redis.NewClient(&redis.Options{
    Addr:         cfg.RedisURL,
    PoolSize:     100,
    MinIdleConns: 10,
})
```

### Database Query Optimization
- Indexed columns: `customer_id`, `payment_id`, `checked_at`
- Connection pooling enabled
- Prepared statements for repeated queries

### NATS Performance
- Asynchronous message handling
- No message queuing delays
- Automatic reconnection

## Security Considerations

- No sensitive data logged
- Redis keys expire automatically
- Database credentials via environment variables
- Rate limiting on HTTP endpoints
- Input validation on all requests

## Graceful Shutdown

The service supports graceful shutdown:
- Completes in-flight fraud checks
- Closes NATS connection
- Closes database connections
- Flushes metrics and traces
- 5-second timeout

## Future Enhancements

Potential improvements:
- Machine learning-based fraud detection
- Real-time model updates
- Customer behavior profiling
- Geographic IP verification
- Device fingerprinting
- Graph-based fraud rings detection

## License

Copyright © 2026
