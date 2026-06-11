BINARY=nebula
BUILD_DIR=bin
GOFLAGS=-ldflags="-s -w"

.PHONY: build run test clean lint docker-build

build:
	go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/nebula

run: build
	./$(BUILD_DIR)/$(BINARY)

test:
	go test -v -race -count=1 ./...

clean:
	rm -rf $(BUILD_DIR)

lint:
	go vet ./...

docker-build:
	docker build -t nebula:latest -f docker/Dockerfile .
