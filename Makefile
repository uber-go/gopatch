export GO111MODULE=on

.PHONY: all
all: build test

.PHONY: build
build:
	go build ./...

.PHONY: test
test:
	go test -v ./...

.PHONY: docker-ci
docker-ci:
	docker run "$$(docker build -q .)"

.PHONY: ci
ci: build test