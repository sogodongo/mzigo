FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o masking ./cmd/masking

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -S mzigo && adduser -S mzigo -G mzigo
WORKDIR /app
COPY --from=builder /build/masking .
USER mzigo
EXPOSE 8084 9104
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s \
    CMD wget -qO- http://localhost:8084/healthz || exit 1
ENTRYPOINT ["./masking"]
