## db/psql: connect to the database using psql
.PHONY: db/psql
db/psql:
	docker compose -f ./compose.yaml exec postgres psql -U mikhaeris -d postgres

.PHONY:
compose/no_cache_up:
	docker compose build --no-cache
	docker compose up -d --force-recreate

## compose/restart name=$1: rebuild without cache and restart container with name=$1
.PHONY: compose/restart
compose/restart:
	docker compose build --no-cache ${name}
	docker compose up -d --force-recreate ${name}

## proto/generate: regenerate Go bindings and OpenAPI from shared contracts
.PHONY: proto/generate
proto/generate:
	$(MAKE) -C proto generate

## services/update_grpc: refresh local contracts and module dependencies
.PHONY: services/update_grpc
services/update_grpc: proto/generate
	cd proto && go mod tidy
	cd auth_service && go mod tidy
	cd customer_service && go mod tidy
	cd notification_service && go mod tidy
	cd gateway && go mod tidy
