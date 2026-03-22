.PHONY: css css-watch build dev run test e2e e2e-up e2e-down e2e-coverage clean htmx-update sqlc

css:
	npx @tailwindcss/cli -i static/css/input.css -o static/css/styles.css --minify

css-watch:
	npx @tailwindcss/cli -i static/css/input.css -o static/css/styles.css --watch

build: css
	go build -o bin/convoke ./cmd/server

dev: dev-infra css
	npx @tailwindcss/cli -i static/css/input.css -o static/css/styles.css --watch &
	CSS_PID=$$!; \
	trap 'kill $$CSS_PID 2>/dev/null' EXIT INT TERM; \
	go run github.com/air-verse/air@latest

dev-infra:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d postgres mock-oauth2-proxy

dev-down:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml down

run:
	go run ./cmd/server

test:
	go test ./...

e2e: e2e-up
	cd e2e && npx playwright test; TEST_EXIT=$$?; \
	$(MAKE) e2e-coverage; \
	$(MAKE) e2e-down; \
	exit $$TEST_EXIT

e2e-up:
	mkdir -p coverage
	docker compose -f docker-compose.test.yml up -d --build --wait

e2e-down:
	docker compose -f docker-compose.test.yml down -v
	rm -rf coverage

e2e-coverage:
	go tool covdata textfmt -i=./coverage -o=coverage.out
	go tool cover -func=coverage.out

clean:
	rm -rf bin/ static/css/styles*.css coverage/ coverage.out

sqlc:
	sqlc generate

htmx-update:
	curl -sL https://unpkg.com/htmx.org@2.0.8/dist/htmx.min.js -o static/js/htmx.min.js
