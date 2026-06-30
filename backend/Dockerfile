# Build Stage
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app

# Copy go.mod and go.sum from the backend directory
COPY backend/go.mod backend/go.sum ./

# Download dependencies (sem vendor/ — D-03, Pitfall 3)
RUN go mod download

# Copy the rest of the backend source code
COPY backend/ .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -v -ldflags="-w -s" -o server .

# Run Stage
FROM alpine:latest
# ca-certificates: necessário para TLS (Oracle, SMTP)
RUN apk add --no-cache ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .
COPY --from=builder /app/migrations ./migrations
EXPOSE 8085
CMD ["./server"]
