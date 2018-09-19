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
	go build -o ./bin/maxfuzz ./cmd/maxfuzz

install:
	mv -t ${GOBIN} ./bin/maxfuzz

teardown:
	docker-compose down

deploy: all
	MAXFUZZ_OPTIONS=storageSolution=local:suppressFuzzerOutput=0 ./bin/maxfuzz
# CLI stuff

tools: build-tools install-tools

build-tools:
	go build -o ./bin/maxfuzz-tools ./cmd/maxfuzz-tools

install-tools:
	mv ./bin/maxfuzz-tools ${GOBIN}/maxfuzz-tools