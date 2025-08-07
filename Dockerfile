# --- Tahap 1: Build Stage ---
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Kompilasi semua binary ke direktori /app/bin
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/migrator ./cmd/migrator
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/worker ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/webapp ./cmd/webapp


# --- Tahap 2: Final Stage ---
FROM alpine:latest

WORKDIR /app

# Salin semua binary dari tahap builder
COPY --from=builder /app/bin /app/bin

# Buat direktori untuk gambar yang di-cache
RUN mkdir -p /app/web/img/channels

# Expose port
EXPOSE 8080

# Perintah default (bisa di-override oleh docker-compose)
CMD ["/app/bin/webapp"]
