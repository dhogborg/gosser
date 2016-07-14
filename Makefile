
.PHONY: build lint clean package

VERSION = $(shell git describe --always --dirty)
TIMESTAMP = $(shell git show -s --format=%ct)

default: build_darwin

build_darwin: 
	GOOS=darwin GOARCH=amd64 go build -a -o ./build/gosser *.go

build_linux: 
	GOOS=linux GOARCH=amd64 go build -a -o ./build/gosser *.go

package: build_linux
	docker build -t dhogborg/gosser:latest .
    
push:
    docker push dhogborg/gosser:latest
	
lint:
	golint .

clean:
	- rm -r build

