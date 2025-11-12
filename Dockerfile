# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make protobuf protobuf-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the applications
RUN go build -o /app/bin/api cmd/api/main.go
RUN go build -o /app/bin/grpc cmd/grpc/main.go

# Runtime stage for API
FROM alpine:latest AS api

WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/bin/api .

EXPOSE 8080

CMD ["./api"]

# Runtime stage for gRPC
FROM alpine:latest AS grpc

WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/bin/grpc .

EXPOSE 50051

CMD ["./grpc"]
