
.PHONY: setup build resources lint clean

VERSION = $(shell git describe --always --dirty)
TIMESTAMP = $(shell git show -s --format=%ct)

default: build_darwin

build_darwin: resources
	GOOS=darwin GOARCH=amd64 go build -a -o ./build/gosser *.go

build_linux: resources
	GOOS=linux GOARCH=amd64 go build -a -o ./build/gosser *.go

lint:
	golint .

clean:
	- rm -r build

