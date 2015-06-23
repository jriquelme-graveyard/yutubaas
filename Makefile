all: build
build:
	goimports -w *.go
	go build
test:
	go test -test.v
cover:
	go test -coverprofile=cover.out
	go tool cover -html=cover.out -o coverage.html
get-tools:
	go get golang.org/x/tools/cmd/goimports
	go get -u github.com/golang/lint/golint

