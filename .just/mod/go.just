devtool := "go tool devtool"

test:
    go test --count=1 -failfast ./...

gen:
    {{ devtool }} gen --all ./internal/cmd/algo

gen-force:
    {{ devtool }} gen --all --force ./internal/cmd/algo

dep:
    go mod tidy

update:
    go get -v -u ./internal/...

fmt:
    {{ devtool }} fmt -w -l .
