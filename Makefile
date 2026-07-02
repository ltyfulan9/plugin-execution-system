.PHONY: test race run demo verify fmt enterprise-schema-check

fmt:
	gofmt -w cmd internal sdk examples

test:
	go test ./...

race:
	go test -race ./...

run:
	METADATA_STORE=local-json ALLOW_LOCAL_JSON_STORE=true go run ./cmd/server

demo:
	python3 scripts/demo_flow.py

verify:
	go test ./...
	python3 scripts/verify_closed_loop.py

enterprise-schema-check:
	test -f migrations/postgres/001_enterprise_metadata.sql
