all: build
build:
	goimports -w *.go
	go build
test:
	go test -test.v
cover:
	go test -coverprofile=cover.out
	sed -i.bak 's/_\/home\/jriquelme\/r\/yutubaas/github.com\/jriquelme\/yutubaas/g' cover.out
	go tool cover -html=cover.out -o coverage.html
get-tools:
	go get golang.org/x/tools/cmd/goimports
	go get -u github.com/golang/lint/golint

