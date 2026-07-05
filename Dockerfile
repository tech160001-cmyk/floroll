# Сборка
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o server ./cmd/server

# Запуск
FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app

RUN mkdir -p /app/data

COPY --from=builder /app/server .
COPY migrations ./migrations
COPY web ./web

ENV FLOROLL_ROOT=/app
ENV PORT=8080
ENV DATABASE_PATH=/app/data/floroll.db

EXPOSE 8080

CMD ["./server"]
