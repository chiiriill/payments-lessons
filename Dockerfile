FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/payments ./cmd/app

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
COPY --from=build /bin/payments /bin/payments
ENTRYPOINT ["/bin/payments"]
