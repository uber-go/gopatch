export GO111MODULE=on

.PHONY:
all: build test

.PHONY:
build:
	go build ./...

.PHONY:
test:
	go test -v ./...
