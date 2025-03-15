.PHONY: migrate-up migrate-down new-migration


run-down:
	docker compose down

run-up:
	docker compose up -d

restart-db:
	docker compose down && rm -rf ./.data && docker compose up -d

log-p:
	docker logs --details --follow --timestamps --tail=1000 inkme-dev-postgres

log-r:
	docker logs --details --follow --timestamps --tail=1000 inkme-dev-redis

run-prom:
	prometheus --config.file=config/prometheus.yml

go-lint: ## Runs linter for .go files
	@golangci-lint run --config ./config/go.yml
	@echo "Go lint passed successfully"

go-pprof:
	go tool pprof http://localhost:6060/debug/pprof/profile

update:
	go get -u

down-dev:
	docker compose down
	rm -rf .data

# Create a new migration file
new-migration:
	@read -p "Enter tenant database name (e.g., tattoo_studio_1): " db; \
	read -p "Enter migration name: " name; \
	migrate create -ext sql -dir migrations/$$db -seq $$name

# Run migrations up
migrate-up:
	@read -p "Enter tenant database name (e.g., tattoo_studio_1): " db; \
	migrate -path migrations/$$db -database "postgres://postgres:password@localhost:5438/$$db?sslmode=disable" up

# Run migrations down
migrate-down:
	@read -p "Enter tenant database name (e.g., tattoo_studio_1): " db; \
	migrate -path migrations/$$db -database "postgres://postgres:password@localhost:5438/$$db?sslmode=disable" down

# Run migrations up for all tenants
migrate-up-all:
	migrate -path migrations/tattoo_studio_1 -database "postgres://postgres:password@localhost:5438/tattoo_studio_1?sslmode=disable" up
	migrate -path migrations/tattoo_studio_2 -database "postgres://postgres:password@localhost:5438/tattoo_studio_2?sslmode=disable" up
