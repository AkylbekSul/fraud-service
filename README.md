# Fraud Service

The Fraud Service is a microservice responsible for real-time fraud detection and risk assessment for payment transactions. It uses rule-based checks and velocity analysis to identify potentially fraudulent activities.

## Overview

The Fraud Service provides:
- Real-time fraud detection for payment transactions
- Risk score calculation based on amount thresholds
- Velocity checks using Redis for rate limiting
- Fraud statistics and analytics
- Dual protocol support: HTTP REST and gRPC
- Historical fraud data storage in PostgreSQL

## Architecture

```
Client / Payment Orchestrator
        ↓                ↓
   HTTP (Gin)        gRPC Server
        ↓                ↓
     Fraud Checker Service
        ↓            ↓
  Redis (velocity)  PostgreSQL (fraud logs)
```

## Technology Stack

- **Language**: Go 1.22
- **Web Framework**: Gin
- **Database**: PostgreSQL (lib/pq)
- **Cache**: Redis (go-redis v9)
- **RPC**: gRPC
- **Observability**: OpenTelemetry + Jaeger + Prometheus
- **Logging**: Zap (Uber)

## Configuration

The service uses environment variables for configuration:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8083` |
| `GRPC_PORT` | gRPC server port | `50052` |
| `DATABASE_URL` | PostgreSQL connection string | Required |
| `REDIS_URL` | Redis server address | Required |
| `JAEGER_ENDPOINT` | Jaeger collector endpoint | `jaeger:4318` |

## API Endpoints

### Fraud Check
```http
POST /fraud/check
```

**Request**:
```json
{
  "payment_id": "payment-uuid",
  "amount": 100.50,
  "customer_id": "customer-123"
}
```

**Response**: `200 OK`
```json
{
  "decision": "approve",
  "reason": "Transaction approved"
}
```

Possible decisions: `approve`, `deny`, `manual_review`

### Get Fraud Statistics
```http
GET /fraud/stats
```

**Response**: `200 OK`
```json
{
  "total_checks": 1500,
  "approved_count": 1400,
  "denied_count": 45,
  "manual_review_count": 55,
  "avg_risk_score": 25.3
}
```

### Health Check
```http
GET /health
```

### Prometheus Metrics
```http
GET /metrics
```

## gRPC Service

The service exposes a gRPC interface on port 50052:

```protobuf
service FraudService {
  rpc CheckFraud(CheckFraudRequest) returns (CheckFraudResponse);
}

message CheckFraudRequest {
  string payment_id = 1;
  double amount = 2;
  string customer_id = 3;
}

message CheckFraudResponse {
  string decision = 1;
  string reason = 2;
}
```

## Fraud Detection Rules

### 1. High Amount Check
- Transactions over **$10,000** are denied

### 2. Velocity Check (Redis-based)
- Maximum **5 transactions per hour** per customer

### 3. Manual Review Threshold
- Transactions over **$5,000** are flagged for manual review

## Risk Score Calculation

Risk scores range from **0 to 100**:

| Condition | Points |
|-----------|--------|
| Amount > $1,000 | +30 |
| Amount > $5,000 | +50 |

## Database Schema

The service uses PostgreSQL with the following tables:

### fraud_decisions
Stores all fraud check results:
- `payment_id`, `customer_id` (indexed)
- `amount`, `decision`, `reason`
- `risk_score`, `created_at`

### fraud_rules
Configurable fraud detection rules:
- `name`, `description`
- `max_amount`, `max_per_hour`
- `active`, `priority`

### velocity_counters
Backup velocity tracking (Redis is primary):
- `entity_type`, `entity_id`
- `counter_type`, `count`
- `window_start`, `window_end`

## Project Structure

```
fraud-service/
├── cmd/
│   └── main.go                      # Application entry point
├── internal/
│   ├── api/
│   │   └── router.go                # HTTP routing setup
│   ├── config/
│   │   └── config.go                # Configuration management
│   ├── grpcserver/
│   │   └── fraud_grpc_server.go     # gRPC service implementation
│   ├── handlers/
│   │   └── fraud_handler.go         # HTTP request handlers
│   ├── interfaces/
│   │   └── fraud_repository.go      # Repository interface
│   ├── models/
│   │   └── fraud.go                 # Domain models
│   ├── repository/
│   │   └── fraud_repository.go      # Data access layer
│   ├── service/
│   │   └── fraud_checker.go         # Fraud detection logic
│   └── telemetry/
│       └── telemetry.go             # OpenTelemetry + Zap setup
├── migrations/
│   └── 001_fraud_service_schema.sql # Database schema
├── Dockerfile
├── go.mod
└── README.md
```

## Running the Service

### Local Development

1. Set up environment variables:
```bash
export DATABASE_URL="postgresql://user:password@localhost:5432/fraud_service?sslmode=disable"
export REDIS_URL="localhost:6379"
export PORT="8083"
export GRPC_PORT="50052"
```

2. Run database migrations:
```bash
psql $DATABASE_URL -f migrations/001_fraud_service_schema.sql
```

3. Start the service:
```bash
go run cmd/main.go
```

### Docker

```bash
docker build -t fraud-service .
docker run -p 8083:8083 -p 50052:50052 \
  -e DATABASE_URL="..." \
  -e REDIS_URL="..." \
  fraud-service
```

## Dependencies

### External Services
- **PostgreSQL**: Fraud check history and analytics
- **Redis**: Velocity tracking and rate limiting
- **Jaeger** (optional): Distributed tracing

## Observability

- **Tracing**: OpenTelemetry with Jaeger exporter (OTLP HTTP)
- **Metrics**: Prometheus endpoint at `/metrics`
- **Logging**: Structured JSON logging via Zap with trace ID correlation (`X-Trace-ID` header)

## Connection Pooling

- **PostgreSQL**: 30 max open, 15 max idle, 5min lifetime
- **Redis**: Pool size 100

## Graceful Shutdown

The service handles OS signals (SIGINT/SIGTERM) with a 5-second timeout, closing database and Redis connections cleanly.

## License

Copyright 2026
