BINARY=agent

.PHONY: build run

build:
	go build -o $(BINARY) ./cmd/agent

run: build
	./$(BINARY)

test: 
	go test ./...

lint:
	golangci-lint run