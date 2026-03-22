default: check

check:
    gofmt -w .
    go test ./...

test:
    go test ./...

test-race:
    go test -race ./...

run:
    go run .

fmt:
    gofmt -w .

vet:
    go vet ./...

tidy:
    go mod tidy
