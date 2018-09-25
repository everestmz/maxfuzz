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

build-dockerfiles:
	docker build -f ./config/docker/Dockerfile_c -t fuzzbox_c .
	docker build -f ./config/docker/Dockerfile_go -t fuzzbox_go .

install:
	mv -t ${GOBIN} ./bin/maxfuzz

teardown:
	docker-compose down

deploy: all
	MAXFUZZ_OPTIONS=storageSolution=local:suppressFuzzerOutput=0:strategy=parallel ./bin/maxfuzz
# CLI stuff

tools: build-tools install-tools

build-tools:
	go build -o ./bin/maxfuzz-tools ./cmd/maxfuzz-tools

install-tools:
	mv ./bin/maxfuzz-tools ${GOBIN}/maxfuzz-tools