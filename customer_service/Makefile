## db/migrations/new name=$1: create a new database migration
.PHONY: db/migration/new
db/migration/new:
	@echo 'Creating migration files for ${name}...'
	migrate create -seq -ext=.sql -dir=./migrations ${name}

## grpc/generate: generate proto contracts
.PHONY: grpc/generate
grpc/generate:
	mkdir -p ./pkg
	protoc \
		--proto_path=./api \
		--proto_path=./third_party \
		--go_out=paths=source_relative:./pkg \
		--go-grpc_out=paths=source_relative:./pkg \
		customer/v1/customer.proto \

	protoc \
		--proto_path=./api \
		--proto_path=./third_party \
		--grpc-gateway_out=paths=source_relative:./pkg \
		--openapi_out=paths=source_relative:. \
		customer/v1/customer.proto
