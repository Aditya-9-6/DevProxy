# Build Stage
FROM golang:alpine AS builder

WORKDIR /app
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o devproxy ./cmd/devproxy

# Final Stage: Minimal scratch / alpine container
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/devproxy /usr/local/bin/devproxy

# Proxy port: 8080, Dashboard port: 8081
EXPOSE 8080 8081

ENTRYPOINT ["devproxy"]
CMD ["-port", "8080", "-web-port", "8081"]
