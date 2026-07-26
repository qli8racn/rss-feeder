# Go 1.25 系イメージは bullseye ベースが提供されていない（MCR上は 2-1.25-bookworm 以降のみ）ため、
# ベースOSも bullseye から bookworm に変更している。
FROM mcr.microsoft.com/devcontainers/go:2-1.25-bookworm

# Install additional tools if needed
# bookworm ベースイメージは /etc/apt/sources.list.d/debian.sources (deb822形式) が
# 唯一の apt ソース定義のため、sources.list.d 配下を全削除すると
# apt-get update 後にリポジトリが空になり sqlite3 が見つからなくなる。
# .sources ファイルは残し、それ以外（旧形式の .list 等）のみ削除する。
RUN find /etc/apt/sources.list.d -mindepth 1 ! -name '*.sources' -delete && \
    apt-get update && apt-get install -y \
    curl \
    gcc \
    libsqlite3-dev \
    sqlite3 \
    && rm -rf /var/lib/apt/lists/*
