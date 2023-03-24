#!make

GOPATH = $(shell go env GOPATH)
GOBIN  = $(GOPATH)/bin
GOX    = go run github.com/mitchellh/gox


.PHONY: build
build:
	CGO_ENABLED=0 go build -v -o ./bin/minirc -ldflags ${LDFLAGS} ./minirc.go

.PHONY: run
run: build
	./bin/minirc
