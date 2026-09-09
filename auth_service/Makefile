.PHONY: confirm
confirm:
	@printf 'Are you sure? [y/N] ' && read ans && [ "$${ans:-N}" = y ]

## db/migrations/new name=$1: create a new database migration
.PHONY: db/migration/new
db/migration/new:
	@echo 'Creating migration files for ${name}...'
	migrate create -seq -ext=.sql -dir=./migrations ${name}

## db/migrations/up: apply all up database migrations
.PHONY: db/migration/up
db/migration/up: confirm
	@echo 'Running up migrations...'
	migrate -path=./migrations -database="postgres://mikhaeris:qwerty@:5432/bank?sslmode=disable" up
