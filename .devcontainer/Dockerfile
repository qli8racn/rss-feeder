FROM mcr.microsoft.com/devcontainers/go:1-1.24-bullseye

# Install additional tools if needed
RUN rm -rf /etc/apt/sources.list.d/* && \
    apt-get update && apt-get install -y \
    curl \
    gcc \
    libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*
