.PHONY: up
up:
	docker compose up -d

.PHONY: logs
logs:
	docker compose logs -f

.PHONY: tests
tests: unit-tests acceptance-tests

.PHONY: unit-tests
unit-tests:
	go test ./...

.PHONY: acceptance-tests
acceptance-tests:
	docker compose exec -T hurl hurl --glob "/tests/**/*.hurl" --test --error-format=long --variable host=caddylura:8082
