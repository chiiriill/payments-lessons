FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy dependency files first to leverage layer caching.
# This layer is rebuilt only when go.mod or go.sum changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# ARG CMD selects which binary to build: app | worker | mock-provider
ARG CMD=app
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /bin/service \
    ./cmd/${CMD}

FROM alpine:3.21

# ca-certificates: TLS connections to payment provider
# tzdata: correct time zone handling in logs
RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /bin/service /service

ENTRYPOINT ["/service"]
