FROM golang:1.24.1 AS apiserver-builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go service
RUN GOOS=linux go build -o apiserver ./cmd/apiserver

FROM node:20.18.0 AS web-builder

ARG VITE_API_BASE_URL=/api
ARG VITE_WS_BASE_URL=/ws
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

FROM nginx:1.27.5-bookworm

WORKDIR /app

# Set default timezone
ENV TZ=UTC

RUN apt-get update && apt-get install -y \
    tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && ln -fs /usr/share/zoneinfo/${TZ} /etc/localtime \
    && echo ${TZ} > /etc/timezone \
    && dpkg-reconfigure -f noninteractive tzdata

# Copy apiserver binary and config
COPY --from=apiserver-builder /app/apiserver /app/
COPY --from=apiserver-builder /app/configs /app/configs
COPY --from=apiserver-builder /app/configs/apiserver.yaml /etc/mcp-gateway/

# Create data directory for apiserver with proper permissions
RUN mkdir -p /app/data && \
    chmod -R 777 /app/data && \
    mkdir -p /tmp/mcp-gateway && \
    chmod -R 777 /tmp/mcp-gateway

# Set minimum required environment variables
ENV APISERVER_I18N_PATH=/app/configs/i18n \
    GATEWAY_DB_NAME=/app/data/mcp-gateway.db

# Copy web files to nginx html directory
COPY --from=web-builder /app/web/dist /usr/share/nginx/html

# Copy nginx configuration
COPY deploy/docker/multi/web/nginx.conf /etc/nginx/nginx.conf

# Expose ports
EXPOSE 80 5234

# Create startup script
RUN echo '#!/bin/bash\n\
nginx &\n\
exec /app/apiserver\n' > /app/start.sh && \
chmod +x /app/start.sh

CMD ["/app/start.sh"] 