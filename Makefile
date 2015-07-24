.PHONY=all build clean

all: build

build:
	go-bindata -pkg maestro files/...
	go build .
	cd client && go build . && cd ..
	mv client/client maestro

clean:
	rm -rf maestro bindata.go
