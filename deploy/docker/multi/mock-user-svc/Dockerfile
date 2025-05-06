FROM golang:1.24.1 AS builder
WORKDIR /app
COPY . .
RUN GOOS=linux go build -o mock-user-svc ./cmd/mock-user-svc

FROM ubuntu:22.04
WORKDIR /app
COPY --from=builder /app/mock-user-svc .

EXPOSE 5236

CMD ["./mock-user-svc"]