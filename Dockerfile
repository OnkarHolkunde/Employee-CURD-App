# ---- Build stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /src

# Cache module downloads separately from source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/server ./cmd/server

# ---- Runtime stage ----
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=builder /out/server .

RUN mkdir -p /app/uploads && chown -R app:app /app
USER app

EXPOSE 8080

ENTRYPOINT ["/app/server"]
