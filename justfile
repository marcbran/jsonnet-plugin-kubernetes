tool := "go tool -modfile=tools/go.mod"

build-jsonnet: vendor-jsonnet
    for dir in pkg/*/; do \
        [ -f "$dir/main.libsonnet" ] || continue; \
        rm -rf "$dir/build" && {{tool}} jpoet pkg build -J "${dir}vendor" "$dir" || exit 1; \
    done

build: build-jsonnet

ci: lint vuln test build test-e2e

lint-go:
    {{tool}} golangci-lint run ./...

lint: lint-go

push-jsonnet: build-jsonnet
    for dir in pkg/*/; do \
        [ -f "$dir/main.libsonnet" ] || continue; \
        {{tool}} jpoet pkg push "$dir" || exit 1; \
    done

push: push-jsonnet

test-e2e:
    go test -v -tags e2e ./tests/...

test-go:
    go test -v -cover -timeout=120s -parallel=10 ./...

test-jsonnet: vendor-jsonnet
    for dir in pkg/*/; do \
        [ -f "$dir/main.libsonnet" ] || continue; \
        {{tool}} jpoet test -J "${dir}vendor" "$dir" || exit 1; \
    done

test: test-go test-jsonnet

vendor-jsonnet:
    for dir in pkg/*/; do \
        [ -f "$dir/jsonnetfile.json" ] || continue; \
        (cd "$dir" && go tool -modfile=../../tools/go.mod jb install) || exit 1; \
    done

vuln-go:
    {{tool}} govulncheck ./...

vuln: vuln-go
