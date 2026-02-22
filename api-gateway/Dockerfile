FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy shared module
COPY shared/ ./shared/

# Copy api-gateway into its own subdirectory so that ../shared resolves correctly
COPY api-gateway/go.mod api-gateway/go.sum* ./api-gateway/

WORKDIR /app/api-gateway
RUN go mod download

COPY api-gateway/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -o api-gateway ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/api-gateway/api-gateway .

EXPOSE 8080

CMD ["./api-gateway"]
