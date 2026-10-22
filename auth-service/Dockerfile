# Build Stage
FROM golang:1.26.8-alpine AS builder

WORKDIR /app

# Install git, make, ca-certificates
RUN apk add --no-cache git make ca-certificates tzdata

# Download dependencies
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /bin/auth-service ./cmd/api

# Production Stage
FROM alpine:3.21

WORKDIR /app

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /bin/auth-service /app/auth-service
COPY --from=builder /app/api /app/api

ENV GIN_MODE=release

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/app/auth-service"]
