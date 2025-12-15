.PHONY: build clean lint install

build:
	go build -trimpath -ldflags="-s -w" -o ai

clean:
	rm -f ai

lint:
	golangci-lint run

install: build
	cp ai $(GOPATH)/bin/ai
