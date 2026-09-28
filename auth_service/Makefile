## db/migrations/new name=$1: create a new database migration
.PHONY: db/migration/new
db/migration/new:
	@echo 'Creating migration files for ${name}...'
	migrate create -seq -ext=.sql -dir=./migrations ${name}

## grpc/generate: regenerate shared RPC contracts and OpenAPI
.PHONY: grpc/generate
grpc/generate:
	$(MAKE) -C ../proto generate/rpc generate/openapi

## services/update_grpc: refresh local contracts and module dependencies
.PHONY: services/update_grpc
services/update_grpc: grpc/generate
	go mod tidy
