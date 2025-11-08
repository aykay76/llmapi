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

dist:
	rm C:\Users\alank\go\bin\agent.exe
	go build -o agent.exe .\cmd\agent\main.go
	mv agent.exe C:\Users\alank\go\bin