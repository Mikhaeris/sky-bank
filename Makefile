## db/psql: connect to the database using psql
.PHONY: db/psql
db/psql:
	docker compose -f ./compose.yaml exec postgres psql -U mikhaeris -d postgres

.PHONY: compose/up
compose/up:
	docker compose up -d

.PHONY: compose/down
compose/down:
	docker compose down

.PHONY:
compose/no_cache_up:
	docker compose build --no-cache
	docker compose up -d --force-recreate

.PHONY:
compose/ps:
	docker compose ps

## compose/restart name=$1: rebuild without cache and restart container with name=$1
.PHONY: compose/restart
compose/restart:
	docker compose build --no-cache ${name}
	docker compose up -d --force-recreate ${name}

## services/update_grpc COMMIT=<commit>: update grpc dependencies to commit
.PHONY: services/update_grpc
services/update_grpc:
ifndef COMMIT
	$(error COMMIT is required. Usage: make services/update_grpc COMMIT=<commit>)
endif
	cd auth_service && \
		go get github.com/mikhaeris/sky-bank/notification_service/pkg/notification/v1@$(COMMIT) && \
		go get github.com/mikhaeris/sky-bank/customer_service/pkg/customer/v1@$(COMMIT) && \
		go mod tidy

	cd customer_service && \
	    go get github.com/mikhaeris/sky-bank/auth_service/pkg/auth/v1@$(COMMIT) && \
		go get github.com/mikhaeris/sky-bank/notification_service/pkg/notification/v1@$(COMMIT) && \
		go mod tidy

	cd gateway && \
        go get github.com/mikhaeris/sky-bank/auth_service/pkg/auth/v1@$(COMMIT) && \
		go get github.com/mikhaeris/sky-bank/customer_service/pkg/customer/v1@$(COMMIT) && \
		go mod tidy
