.PHONY: run test build

run:
	go run ./cmd/spotself

test:
	go test ./...

build:
	go build -o spotself ./cmd/spotself
	go build -o spotselfctl ./cmd/spotselfctl
