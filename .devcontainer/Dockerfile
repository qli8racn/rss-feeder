FROM mcr.microsoft.com/devcontainers/go:1-1.24-bullseye

# Install additional tools if needed
RUN rm -rf /etc/apt/sources.list.d/* && \
    apt-get update && apt-get install -y \
    curl \
    gcc \
    libsqlite3-dev \
    sqlite3 \
    cron \
    && rm -rf /var/lib/apt/lists/*

# RSSフィードの定期取得・要約用cronジョブ（コンテナ起動時にpostStartCommandで開始する）
COPY poll-feeds.cron /etc/cron.d/poll-feeds
RUN chmod 0644 /etc/cron.d/poll-feeds
