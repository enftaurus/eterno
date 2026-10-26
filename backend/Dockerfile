# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/eterno-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/eterno-worker ./cmd/worker

# API target
FROM alpine:3.19 AS api
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /bin/eterno-api /app/eterno-api
COPY migrations /app/migrations
EXPOSE 8080
ENTRYPOINT ["/app/eterno-api"]

# Worker target
FROM alpine:3.19 AS worker
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /bin/eterno-worker /app/eterno-worker
ENTRYPOINT ["/app/eterno-worker"]
