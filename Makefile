# Maxfuzz

all: dep clean build

dep:
	dep ensure

clean:
	@rm -rf ./bin

test:
	@echo "=============="
	@echo "== UNIT TESTS:"
	MAXFUZZ_ENV="test" go test ./internal/helpers -v -tags=unit
	@echo "=============="

build:
	# go build -o ./bin/monitor ./cmd/monitor
	# go build -o ./bin/go-monitor ./cmd/go-monitor
	# go build -o ./bin/pre-sync ./cmd/pre-sync
	# go build -o ./bin/reproduce ./cmd/reproduce
	go build -o ./bin/maxfuzz ./cmd/maxfuzz

install:
	mv -t ${GOBIN} ./bin/maxfuzz

teardown:
	docker-compose down

deploy: teardown
	SYNC_DIR=$(shell pwd)/sync docker-compose build
	SYNC_DIR=$(shell pwd)/sync docker-compose up
# CLI stuff

tools: build-tools install-tools

build-tools:
	go build -o ./bin/maxfuzz-tools ./cmd/maxfuzz-tools

install-tools:
	mv ./bin/maxfuzz-tools ${GOBIN}/maxfuzz-tools