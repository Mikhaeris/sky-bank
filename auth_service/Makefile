## db/migrations/new name=$1: create a new database migration
.PHONY: db/migration/new
db/migration/new:
	@echo 'Creating migration files for ${name}...'
	migrate create -seq -ext=.sql -dir=./migrations ${name}

## grpc/generate: genereate proto contract
.PHONY: grpc/generate
grpc/generate:
	protoc \
		--proto_path=./api \
		--proto_path=./third_party \
		--go_out=paths=source_relative:./api \
		--go-grpc_out=paths=source_relative:./api \
		--grpc-gateway_out=paths=source_relative:./api \
		--openapi_out==paths=source_relative:. \
		auth/v1/auth.proto
