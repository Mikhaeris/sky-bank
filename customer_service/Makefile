## db/migrations/new name=$1: create a new database migration
.PHONY: db/migration/new
db/migration/new:
	@echo 'Creating migration files for ${name}...'
	migrate create -seq -ext=.sql -dir=./migrations ${name}

## grpc/generate: generate proto contracts
.PHONY: grpc/generate
grpc/generate:
	protoc \
		--proto_path=./api \
		--proto_path=./third_party \
		--go_out=paths=source_relative:./api \
		--go-grpc_out=paths=source_relative:./api \
		customer/v1/customer.proto \
		customer/v1/customer_private.proto

	protoc \
		--proto_path=./api \
		--proto_path=./third_party \
		--grpc-gateway_out=paths=source_relative:./api \
		--openapi_out=paths=source_relative:. \
		customer/v1/customer.proto
