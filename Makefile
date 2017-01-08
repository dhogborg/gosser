
.PHONY: build lint clean package

VERSION = $(shell git describe --always --dirty)
TIMESTAMP = $(shell git show -s --format=%ct)

default: build

build:
	go build -a -o ./build/gosser *.go

install:
	go install .

build_darwin:
	GOOS=darwin GOARCH=amd64 go build -a -o ./build/gosser *.go
	zip ./build/gosser_darwin64.zip ./build/gosser

build_linux:
	GOOS=linux GOARCH=amd64 go build -a -o ./build/gosser *.go
	zip ./build/gosser_linux64.zip ./build/gosser

build_arm5:
	GOOS=linux GOARM=5 GOARCH=arm go build -a -o ./build/gosser *.go
	zip ./build/gosser_linux_arm5.zip ./build/gosser

build_arm7:
	GOOS=linux GOARM=7 GOARCH=arm go build -a -o ./build/gosser *.go
	zip ./build/gosser_linux_arm7.zip ./build/gosser

build_win64:
	GOOS=windows GOARCH=amd64 go build -a -o ./build/gosser.exe *.go
	zip ./build/gosser_win64.zip ./build/gosser.exe

build_win32:
	GOOS=windows GOARCH=386 go build -a -o ./build/gosser.exe *.go
	zip ./build/gosser_win32.zip ./build/gosser.exe

all: build_darwin build_linux build_arm5 build_arm7 build_win64 build_win32
	rm ./build/gosser
	rm ./build/gosser.exe

package: build_linux
	docker build -t dhogborg/gosser:latest .
    
push:
    docker push dhogborg/gosser:latest
	
	
lint:
	golint .

clean:
	- rm -r build

