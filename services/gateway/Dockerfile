# Multi-stage build. The builder stage has the full Go toolchain.
# The final image has only the compiled binary and ca-certificates.
# Result: ~15MB image vs ~800MB with the full toolchain.

FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s" \
    -o gateway \
    ./cmd/gateway

FROM alpine:3.19

# ca-certificates is required for TLS connections to the contracts service
# and Schema Registry in production.
RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S mzigo && adduser -S mzigo -G mzigo

WORKDIR /app
COPY --from=builder /build/gateway .

USER mzigo

EXPOSE 8080 9100

HEALTHCHECK --interval=10s --timeout=3s --start-period=15s \
    CMD wget -qO- http://localhost:8080/healthz || exit 1

ENTRYPOINT ["./gateway"]
