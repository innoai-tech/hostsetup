test:
    go test --count=1 -failfast ./...

dep:
    go mod tidy

update:
    go get -v -u ./internal/...

fmt:
    go tool fmt .

gen:
    go generate -v ./...
