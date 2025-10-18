.PHONY: build

build:
	go build -v ./cmd/userservice

.PHONY: run

run: build
	./userservice --config=./config/local.yaml

.PHONY: migrate

migrate:
	go run ./cmd/migrator \
	 --storage-path=./storage/userservice.db \
	 --migrations-path=./migrations

.PHONY: test_migrate

test_migrate:
	go run ./cmd/migrator \
	 --storage-path=./storage/userservice.db \
	 --migrations-path=./tests/migrations \
	 --migrations-table=migrations_test

.DEFAULT_GOAL := run