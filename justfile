tool := "go tool -modfile=tools/go.mod"

test-go:
    go test -v -cover -timeout=120s -parallel=10 ./...

test-jsonnet:
    {{tool}} jpoet test .

test: test-go test-jsonnet

lint-go:
    {{tool}} golangci-lint run ./...

lint: lint-go

vuln-go:
    {{tool}} govulncheck ./...

vuln: vuln-go

build:
    {{tool}} jpoet pkg build

push: build
    {{tool}} jpoet pkg push

test-e2e:
    go test -v -tags e2e ./tests/...

ci: lint vuln test build test-e2e
