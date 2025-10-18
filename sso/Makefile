.PHONY: build

build:
	go build -v ./cmd/sso

.PHONY: run

run: build
	./sso --config=./config/local.yaml

.PHONY: migrate

migrate:
	go run ./cmd/migrator \
	 --storage-path=./storage/sso.db \
	 --migrations-path=./migrations

.PHONY: test_migrate

test_migrate:
	go run ./cmd/migrator \
	 --storage-path=./storage/sso.db \
	 --migrations-path=./tests/migrations \
	 --migrations-table=migrations_test

.DEFAULT_GOAL := run