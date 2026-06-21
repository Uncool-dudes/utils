lint:
    golangci-lint run

test:
    go test ./...

vuln:
    go run golang.org/x/vuln/cmd/govulncheck@latest -scan=package ./...

ci: lint test vuln
