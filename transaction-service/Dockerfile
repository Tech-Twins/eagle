FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy shared module
COPY shared/ /app/shared/

# Copy transaction service
COPY transaction-service/go.mod transaction-service/go.sum* ./
RUN go mod download

COPY transaction-service/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -o transaction-service ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/transaction-service .

EXPOSE 8084

CMD ["./transaction-service"]
