.PHONY=all build clean test

all: clean build

build:
	go-bindata -pkg maestro files/...
	go build .
	cd client && go build . && cd ..
	mv client/client maestro

clean:
	rm -rf maestro bindata.go

test:
	cd tests && go test && cd ..
