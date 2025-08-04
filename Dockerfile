# --- Tahap 1: Build Stage ---
# Menggunakan image Go resmi sebagai dasar untuk kompilasi
FROM golang:1.24-alpine AS builder

# Menetapkan direktori kerja di dalam container
WORKDIR /app

# Menyalin file go.mod dan go.sum untuk mengunduh dependensi
COPY go.mod go.sum ./
RUN go mod download

# Menyalin seluruh kode sumber aplikasi
COPY . .

# Melakukan kompilasi untuk setiap modul aplikasi
# -o menentukan nama output binary
# -ldflags="-w -s" untuk mengurangi ukuran binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/migrator ./cmd/migrator
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/worker ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/webapp ./cmd/webapp


# --- Tahap 2: Final Stage ---
# Menggunakan image Alpine Linux yang sangat ringan sebagai dasar
FROM alpine:latest

# Menyalin binary yang sudah dikompilasi dari tahap 'builder'
COPY --from=builder /bin/migrator /bin/migrator
COPY --from=builder /bin/worker /bin/worker
COPY --from=builder /bin/webapp /bin/webapp

# Menyalin template HTML dan file .env
COPY web/templates /web/templates
COPY .env .env

# Expose port yang akan digunakan oleh webapp (nilai default)
# Port ini bisa di-override oleh Docker Compose
EXPOSE 8080

# Perintah default yang akan dijalankan saat container dimulai
# Menjalankan worker di background dan webapp di foreground
CMD sh -c "/bin/worker & /bin/webapp"
