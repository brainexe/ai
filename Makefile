.PHONY: build clean lint

build:
	go build -ldflags="-s -w" -o ai

clean:
	rm -f ai

lint:
	golangci-lint run
