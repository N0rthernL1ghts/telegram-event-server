.PHONY: build clean

PROJECT_NAME := TelegamEventServer
BINARY_NAME := tg-events-service
SERVICE_DIR := ./server
BIN_DIR := ./build/release

build:
	go build -o $(BIN_DIR)/$(BINARY_NAME) $(SERVICE_DIR)

clean:
	rm -rf $(BIN_DIR)

run: build
	$(BIN_DIR)/$(BINARY_NAME)
