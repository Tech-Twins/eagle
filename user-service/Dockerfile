FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy shared module
COPY shared/ ./shared/

# Copy user service into its own subdirectory so that ../shared resolves correctly
COPY user-service/go.mod user-service/go.sum* ./user-service/

WORKDIR /app/user-service
RUN go mod download

COPY user-service/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -o user-service ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/user-service/user-service .

EXPOSE 8082

CMD ["./user-service"]
