.PHONY: css css-watch build dev run test e2e e2e-up e2e-down e2e-coverage clean htmx-update

css:
	npx @tailwindcss/cli -i static/css/input.css -o static/css/styles.css --minify

css-watch:
	npx @tailwindcss/cli -i static/css/input.css -o static/css/styles.css --watch

build: css
	go build -o bin/convoke ./cmd/server

dev:
	npx @tailwindcss/cli -i static/css/input.css -o static/css/styles.css --watch &
	DEV_MODE=true go run ./cmd/server

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

htmx-update:
	curl -sL https://unpkg.com/htmx.org@2.0.8/dist/htmx.min.js -o static/js/htmx.min.js
