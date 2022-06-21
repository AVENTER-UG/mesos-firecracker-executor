#Dockerfile vars

#vars
TAG=dev
BUILDDATE=`date -u +%Y-%m-%dT%H:%M:%SZ`
BRANCH=`git symbolic-ref --short HEAD`
UID=`id -u`
GID=`id -g`

.PHONY: help build all docs 

help:
	    @echo "Makefile arguments:"
	    @echo ""
	    @echo "Makefile commands:"
			@echo "push"
	    @echo "build"
	    @echo "all"
			@echo "docs"

.DEFAULT_GOAL := all

build:
	@echo ">>>> Build binary"
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-s -w -X main.BuildVersion=${BUILDDATE} -X main.GitVersion=${TAG} -X main.VersionURL=${VERSION_URL} -extldflags \"-static\"" .

build-debian:
	@echo ">>>> Build binary for almalinux8"
	@export DOCKER_CONTEXT=default; docker run --rm -e PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin -e GOPATH=/tmp -e GOCACHE=/tmp -e CGO_ENABLED=0 -e GOOS=linux -u ${UID}:${GID} -w /data -v ${PWD}:/data avhost/debian_build:11 go build -o /data/mesos-firecracker-executor . 
	@mv mesos-firecracker-executor http/

docs:
	@echo ">>>> Build docs"
	$(MAKE) -C $@

all: build
