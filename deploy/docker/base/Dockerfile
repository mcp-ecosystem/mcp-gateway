FROM ubuntu:22.04

ENV TZ=UTC
ENV DEBIAN_FRONTEND=noninteractive

RUN sed -i 's|http://.*.ubuntu.com|http://mirrors.aliyun.com|g' /etc/apt/sources.list && \
    sed -i 's|http://security.ubuntu.com|http://mirrors.aliyun.com|g' /etc/apt/sources.list && \
    sed -i 's|http://ports.ubuntu.com|http://mirrors.aliyun.com|g' /etc/apt/sources.list

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    gnupg

RUN sed -i 's|http://mirrors.aliyun.com|https://mirrors.aliyun.com|g' /etc/apt/sources.list

RUN apt-get update && apt-get install -y --no-install-recommends \
    supervisor \
    nginx \
    tzdata \
    vim \
    python3 \
    python3-pip \
    python3-venv && \
    ln -fs /usr/share/zoneinfo/${TZ} /etc/localtime && \
    echo ${TZ} > /etc/timezone && \
    dpkg-reconfigure -f noninteractive tzdata

RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && \
    apt-get update && apt-get install -y nodejs && \
    npm install -g npm@latest

RUN python3 -m pip install --no-cache-dir uv

RUN apt-get clean && rm -rf \
    /var/lib/apt/lists/* \
    /tmp/* /var/tmp/* \
    /usr/share/doc /usr/share/man /usr/share/info /usr/share/lintian /usr/share/locale

WORKDIR /app

CMD ["bash"]