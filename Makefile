#Dockerfile vars

#vars
TAG=`git describe`
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

build-almalinux8:
	@echo ">>>> Build binary for almalinux8"
	@docker run -e GOPATH=/tmp -e GOCACHE=/tmp -e CGO_ENABLED=0 -e GOOS=linux -u ${UID}:${GID} -w /data -v ${PWD}:/data avhost/almalinux8_rpmbuild go build -o /data/mesos-mainframe-executor . 

docs:
	@echo ">>>> Build docs"
	$(MAKE) -C $@

all: build 
