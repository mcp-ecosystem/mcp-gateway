FROM golang:1.24.1 AS builder
WORKDIR /app
COPY . .
RUN GOOS=linux go build -o mock-user-svc ./cmd/mock-user-svc

FROM ubuntu:22.04
WORKDIR /app

# Set default timezone
ENV TZ=UTC

RUN apt-get update && apt-get install -y \
    tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && ln -fs /usr/share/zoneinfo/${TZ} /etc/localtime \
    && echo ${TZ} > /etc/timezone \
    && dpkg-reconfigure -f noninteractive tzdata

COPY --from=builder /app/mock-user-svc .

EXPOSE 5236

CMD ["./mock-user-svc"]