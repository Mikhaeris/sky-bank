## db/migrations/new name=$1: create a new database migration
.PHONY: db/migration/new
db/migration/new:
	@echo 'Creating migration files for ${name}...'
	migrate create -seq -ext=.sql -dir=./migrations ${name}

## grpc/generate: regenerate shared RPC contracts and OpenAPI
.PHONY: grpc/generate
grpc/generate:
	$(MAKE) -C ../proto generate/rpc generate/openapi
