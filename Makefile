.PHONY=all build clean test

all: clean build

build:
	ln -s config/config.sample maestro.json
	go-bindata -pkg maestro templates
	go build .
	cd cmd && go build . && cd ..
	mv cmd/cmd maestro

clean:
	rm -rf maestro bindata.go maestro.json

test:
	cd tests && go test && cd ..
