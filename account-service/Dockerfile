FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy shared module
COPY shared/ /app/shared/

# Copy account service
COPY account-service/go.mod account-service/go.sum* ./
RUN go mod download

COPY account-service/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -o account-service ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/account-service .

EXPOSE 8083

CMD ["./account-service"]
