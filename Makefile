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
	prometheus --config.file=./configs/prometheus.yml

go-lint: ## Runs linter for .go files
	@golangci-lint run --config ./config/go.yml
	@echo "Go lint passed successfully"
