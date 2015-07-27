# Maestro Makefile
UNAME := $(shell uname -s)

.PHONY=all build clean test install uninstall start-registry stop-registry start-cluster stop-cluster

all: clean build install
build: link build-go
start-registry: build install
start-cluster: build install start-registry

link:
ifeq ($(UNAME), Darwin)
		mkdir -p ${HOME}/.maestro-share/maestro
		mkdir -p ${HOME}/.maestro-share/docker
endif
	ln -s config/maestro-metrics.json maestro.json || echo "continuing..."
	cd vagrant/maestro-vagrant-registry && ln -s user-data.sample user-data || echo "continuing..." && cd -
	cd vagrant/maestro-vagrant-cluster && ln -s user-data.sample user-data || echo "continuing..." && cd -

build-go:
	go-bindata -pkg maestro templates
	go build .
	go build ./cmd/maestro

clean:
	go clean .
	go clean ./cmd/maestro
	rm -rf maestro bindata.go

test:
	go test -v tests

install:
	go install ./cmd/maestro

uninstall:
	rm -f ${GOPATH}/maestro

start-registry:
	@read -p "are you sure you want to start the registry [y/N]? " answer; \
		[ $$answer = "y" ] || [ $$answer = "Y" ] || (echo "exiting..."; exit 1;)
	cd vagrant/maestro-vagrant-registry && vagrant up && vagrant ssh registry

stop-registry:
	@read -p "are you sure you want to stop the registry [y/N]? " answer; \
		[ $$answer = "y" ] || [ $$answer = "Y" ] || (echo "exiting..."; exit 1;)
	cd vagrant/maestro-vagrant-registry && vagrant destroy -f && cd -

start-cluster:
	@read -p "are you sure the registry is up [y/N]? (ssh to registry and run systemctl status registry) " answer; \
		[ $$answer = "y" ] || [ $$answer = "Y" ] || (echo "exiting..."; exit 1;)
	cd vagrant/maestro-vagrant-cluster && vagrant up && vagrant ssh core-03

stop-cluster:
	@read -p "are you sure you want to stop the cluster [y/N]? " answer; \
		[ $$answer = "y" ] || [ $$answer = "Y" ] || (echo "exiting..."; exit 1;)
	cd vagrant/maestro-vagrant-cluster && vagrant destroy -f && cd -

