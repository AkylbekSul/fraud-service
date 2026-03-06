FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy proto module (needed by replace directive in go.mod)
COPY proto/ /app/proto/

# Copy service source
COPY services/fraud-service/ /app/services/fraud-service/

WORKDIR /app/services/fraud-service

RUN go mod download && go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -o fraud-service ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/services/fraud-service/fraud-service .

EXPOSE 8083 50052

CMD ["./fraud-service"]
