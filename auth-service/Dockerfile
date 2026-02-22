FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy shared module first
COPY shared/ /app/shared/

# Copy auth service
COPY auth-service/go.mod auth-service/go.sum* ./
RUN go mod download

COPY auth-service/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -o auth-service ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/auth-service .

EXPOSE 8081

CMD ["./auth-service"]
