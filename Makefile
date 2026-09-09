## db/psql: connect to the database using psql
.PHONY: db/psql
db/psql:
	docker compose -f ./db/docker-compose.yml exec postgres psql -U mikhaeris -d bank
