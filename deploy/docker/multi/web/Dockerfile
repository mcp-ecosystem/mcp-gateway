FROM node:20.18.0 AS builder

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

# Set default timezone
ENV TZ=UTC

RUN apt-get update && apt-get install -y \
    tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && ln -fs /usr/share/zoneinfo/${TZ} /etc/localtime \
    && echo ${TZ} > /etc/timezone \
    && dpkg-reconfigure -f noninteractive tzdata

COPY --from=builder /app/web/dist /usr/share/nginx/html

COPY deploy/docker/multi/nginx.conf /etc/nginx/nginx.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"] 