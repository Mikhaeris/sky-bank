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
		auth/v1/auth.proto \
		auth/v1/challenge.proto

	protoc \
		--proto_path=./api \
		--proto_path=./third_party \
		--grpc-gateway_out=paths=source_relative:./pkg \
		--openapi_out=paths=source_relative:. \
		auth/v1/auth.proto

## services/update_grpc commit=$1: update grpc for commit
.PHONY:
services/update_grpc:
	go get github.com/mikhaeris/sky-bank/notification_service/pkg/notification/v1@${commit}
