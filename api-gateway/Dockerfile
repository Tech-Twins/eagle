FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy shared module
COPY shared/ /app/shared/

# Copy api-gateway
COPY api-gateway/go.mod api-gateway/go.sum* ./
RUN go mod download

COPY api-gateway/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -o api-gateway ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/api-gateway .

EXPOSE 8080

CMD ["./api-gateway"]
