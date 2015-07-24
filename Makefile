.PHONY=all build clean test

all: clean build

build:
	go-bindata -pkg maestro templates
	go build .
	cd cmd && go build . && cd ..
	mv cmd/cmd maestro

clean:
	rm -rf maestro bindata.go

test:
	cd tests && go test && cd ..
