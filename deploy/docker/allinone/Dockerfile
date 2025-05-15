FROM golang:1.24.1 AS builder

WORKDIR /app

# Install SQLite dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    libc6-dev \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN GOOS=linux go build -o /app/bin/apiserver ./cmd/apiserver
RUN GOOS=linux go build -o /app/bin/mcp-gateway ./cmd/mcp-gateway
RUN GOOS=linux go build -o /app/bin/mock-user-svc ./cmd/mock-user-svc

FROM node:20.18.0 AS web-builder

ARG VITE_API_BASE_URL=/api
ARG VITE_WS_BASE_URL=/api/ws
ARG VITE_MCP_GATEWAY_BASE_URL=/mcp
ARG VITE_BASE_URL=/

ENV VITE_API_BASE_URL=${VITE_API_BASE_URL} \
    VITE_WS_BASE_URL=${VITE_WS_BASE_URL} \
    VITE_MCP_GATEWAY_BASE_URL=${VITE_MCP_GATEWAY_BASE_URL} \
    VITE_BASE_URL=${VITE_BASE_URL}

WORKDIR /app/web

COPY web/package*.json ./

RUN npm install

COPY web/ .

RUN npm run build

FROM ubuntu:22.04

# Set default timezone
ENV TZ=UTC

RUN apt-get update && apt-get install -y \
    supervisor \
    nginx \
    curl \
    vim \
    tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && ln -fs /usr/share/zoneinfo/${TZ} /etc/localtime \
    && echo ${TZ} > /etc/timezone \
    && dpkg-reconfigure -f noninteractive tzdata

COPY deploy/docker/allinone/supervisord.conf /etc/supervisor/conf.d/
COPY deploy/docker/allinone/nginx.conf /etc/nginx/nginx.conf
RUN mkdir -p /app/data
COPY configs/apiserver.yaml /etc/mcp-gateway/
COPY configs/mcp-gateway.yaml /etc/mcp-gateway/
COPY configs/i18n /app/configs/i18n/

COPY --from=builder /app/bin/mcp-gateway /usr/local/bin/
COPY --from=builder /app/bin/mock-user-svc /usr/local/bin/
COPY --from=builder /app/bin/apiserver /usr/local/bin/

COPY --from=web-builder /app/web/dist /usr/share/nginx/html

EXPOSE 80

CMD ["/usr/bin/supervisord", "-n", "-c", "/etc/supervisor/conf.d/supervisord.conf"]