FROM golang:1.24.1 AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go service
RUN GOOS=linux go build -o apiserver ./cmd/apiserver

FROM ubuntu:22.04
WORKDIR /app

# Install basic tools
RUN apt-get update && apt-get install -y curl iputils-ping && rm -rf /var/lib/apt/lists/*

# Copy the binary
COPY --from=builder /app/apiserver .

# Copy configs
COPY --from=builder /app/configs /app/configs
RUN mkdir -p /app/data

# Expose port
EXPOSE 5234

# Run the service
CMD ["./apiserver"]