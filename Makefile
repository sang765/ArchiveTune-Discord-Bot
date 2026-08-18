BINARY := bin/discord-forum-bot

.PHONY: test vet build run

test:
	GOTOOLCHAIN=local go test ./...

vet:
	GOTOOLCHAIN=local go vet ./...

build:
	mkdir -p bin
	GOTOOLCHAIN=local go build -trimpath -ldflags='-s -w' -o $(BINARY) ./cmd/bot

run:
	CONFIG_FILE=./config.yaml GOTOOLCHAIN=local go run ./cmd/bot
