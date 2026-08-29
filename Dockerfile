# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.26.5 AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/maint ./main.go

# Runtime stage
FROM debian:bookworm-slim

# Ferramentas externas: cliente postgres 18 (PGDG), minio client, tar
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        gnupg \
        curl \
        ca-certificates \
        lsb-release \
    && curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor -o /usr/share/keyrings/pgdg.gpg \
    && echo "deb [signed-by=/usr/share/keyrings/pgdg.gpg] https://apt.postgresql.org/pub/repos/apt bookworm-pgdg main" > /etc/apt/sources.list.d/pgdg.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
        postgresql-client-18 \
        tar \
        gzip \
    && rm -rf /var/lib/apt/lists/*

# Instala o MinIO Client (mc)
RUN curl -sL https://dl.min.io/client/mc/release/linux-amd64/mc -o /usr/bin/mc \
    && chmod +x /usr/bin/mc

# Instala o docker CLI (client) para o migrate disparar o container redgate/flyway
# contra o daemon do host via socket exposto pelo wrapper.
ARG DOCKER_CLI_VERSION="25.0.5"
RUN curl -fsSL "https://download.docker.com/linux/static/stable/x86_64/docker-${DOCKER_CLI_VERSION}.tgz" -o /tmp/docker.tgz \
    && tar -xzf /tmp/docker.tgz -C /tmp \
    && mv /tmp/docker/docker /usr/bin/docker \
    && rm -rf /tmp/docker.tgz /tmp/docker

COPY --from=builder /out/maint /usr/bin/maint

# Diretório padrão montado pelo wrapper (profiles + backups)
RUN mkdir -p /data/profiles /data/backups

ENTRYPOINT ["maint"]