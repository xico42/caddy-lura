.PHONY: up
up:
	docker compose up -d

.PHONY: logs
logs:
	docker compose logs -f

.PHONY: tests
tests:
	docker compose exec -T hurl hurl --glob "/tests/**/*.hurl" --test --error-format=long