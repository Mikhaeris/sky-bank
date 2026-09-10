## db/psql: connect to the database using psql
.PHONY: db/psql
db/psql:
	docker compose -f ./compose.yaml exec postgres psql -U mikhaeris -d skybankDB

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
