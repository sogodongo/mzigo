FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o contracts ./cmd/contracts

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -S mzigo && adduser -S mzigo -G mzigo
WORKDIR /app
COPY --from=builder /build/contracts .
USER mzigo
EXPOSE 8081 9101
HEALTHCHECK --interval=10s --timeout=3s --start-period=20s \
    CMD wget -qO- http://localhost:8081/healthz || exit 1
ENTRYPOINT ["./contracts"]
