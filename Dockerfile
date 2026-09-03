FROM golang:1.22-bookworm AS builder
WORKDIR /app
COPY . .
RUN go mod download && CGO_ENABLED=0 GOOS=linux go build -o maint .

FROM postgres:18-bookworm
RUN apt-get update && apt-get install -y --no-install-recommends tzdata curl ca-certificates && rm -rf /var/lib/apt/lists/* \
    && curl -fsSL https://dl.min.io/client/mc/release/linux-amd64/mc -o /usr/local/bin/mc \
    && chmod +x /usr/local/bin/mc
WORKDIR /data
COPY --from=builder /app/maint /maint
ENTRYPOINT ["/maint"]
