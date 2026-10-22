# Build Stage
FROM golang:1.26.8-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git make ca-certificates tzdata

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /bin/main-service ./cmd/api

# Production Stage
FROM alpine:3.21

WORKDIR /app

RUN addgroup -S appgroup && adduser -S appuser -G appgroup \
    && mkdir -p /app/assets /tmp/abet-uploads \
    && chown -R appuser:appgroup /app /tmp/abet-uploads

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /bin/main-service /app/main-service
COPY --from=builder /app/api /app/api

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/app/main-service"]
