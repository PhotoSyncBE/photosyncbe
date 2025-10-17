# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary
COPY --from=builder /app/server .

# Copy config
COPY config.yaml .

# Copy Let's Encrypt certificates
# Note: You'll need to copy certs to ./certs/ directory before building
COPY certs/ /root/certs/

# Expose port
EXPOSE 8443

CMD ["./server"]

