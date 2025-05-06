FROM golang:1.24.1 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN GOOS=linux go build -o mcp-gateway ./cmd/mcp-gateway

FROM ubuntu:22.04
WORKDIR /app
RUN apt-get update && apt-get install -y curl iputils-ping && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/mcp-gateway .
COPY --from=builder /app/configs /app/configs
RUN mkdir -p /app/data

EXPOSE 5235
CMD ["./mcp-gateway"]