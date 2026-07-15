.PHONY: run build test vet tidy docker-up docker-down docker-logs clean

APP_NAME := server
BIN_DIR := bin

run:
	go run ./cmd/server

build:
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BIN_DIR)/$(APP_NAME) ./cmd/server

tidy:
	go mod tidy

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v

docker-logs:
	docker compose logs -f app

clean:
	rm -rf $(BIN_DIR)
