# Image stuff

all: build-base deploy-dev

build-base-fresh:
	docker build -t maxfuzz ./fuzzer-base --no-cache

build-base:
	docker build -t maxfuzz ./fuzzer-base

teardown:
	docker-compose -f docker-compose-stable.yml down --remove-orphans
	docker-compose -f docker-compose-dev.yml down --remove-orphans

deploy-stable: teardown
	SYNC_DIR=$(shell pwd)/sync docker-compose -f docker-compose-stable.yml build
	SYNC_DIR=$(shell pwd)/sync docker-compose -f docker-compose-stable.yml up

deploy-dev: teardown
	SYNC_DIR=$(shell pwd)/sync docker-compose -f docker-compose-dev.yml build
	SYNC_DIR=$(shell pwd)/sync docker-compose -f docker-compose-dev.yml up

# CLI stuff

build-cli:
	go build -o ./bin/maxfuzz ./cmd/maxfuzz