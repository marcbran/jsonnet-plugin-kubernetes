
test-go:
    go test -v -cover -timeout=120s -parallel=10 ./...

test-jsonnet:
    jpoet test .

test: test-go test-jsonnet

lint-go:
    golangci-lint run ./...

lint: lint-go

build:
    jpoet pkg build

push: build
    jpoet pkg push

test-e2e:
    go test -v -tags e2e ./tests/...

ci: lint test build test-e2e
