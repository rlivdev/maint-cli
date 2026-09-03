FROM golang:1.22-bookworm AS builder
WORKDIR /app
COPY . .
RUN go mod download && CGO_ENABLED=0 GOOS=linux go build -o maint .

FROM postgres:18-bookworm
RUN apt-get update && apt-get install -y --no-install-recommends tzdata && rm -rf /var/lib/apt/lists/*
WORKDIR /data
COPY --from=builder /app/maint /maint
ENTRYPOINT ["/maint"]
