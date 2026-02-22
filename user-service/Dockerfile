FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy shared module
COPY shared/ /app/shared/

# Copy user service
COPY user-service/go.mod user-service/go.sum* ./
RUN go mod download

COPY user-service/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -o user-service ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/user-service .

EXPOSE 8082

CMD ["./user-service"]
