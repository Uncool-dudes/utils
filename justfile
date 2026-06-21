lint:
    golangci-lint run

test:
    go test ./...

vuln:
    go run golang.org/x/vuln/cmd/govulncheck@latest ./...

ci: lint test vuln
