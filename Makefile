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
	@echo ">>>> Build binary for debian 11"
	@export DOCKER_CONTEXT=default; docker run --rm -e PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin -e GOPATH=/tmp -e GOCACHE=/tmp -e CGO_ENABLED=0 -e GOOS=linux -u ${UID}:${GID} -w /data -v ${PWD}:/data avhost/debian_build:11 go build -o /data/mesos-firecracker-executor . 
	@cp mesos-firecracker-executor http/
	@mv mesos-firecracker-executor build/

copy-docker:
	@echo ">>>> Copy into mesos-mini"
	docker cp build/mesos-firecracker-executor mesos:/usr/libexec/mesos/mesos-firecracker-executor
	docker cp build/rootfs.ext4 mesos:/usr/libexec/mesos/rootfs.ext4
	docker cp build/vmlinux mesos:/usr/libexec/mesos/vmlinux
	docker cp build/firecracker mesos:/usr/local/bin/firecracker
	docker cp resources/fcnet.conflist mesos:/etc/cni/conf.d/fcnet.conflist
	docker cp resources/tc-redirect-tap mesos:/opt/cni/bin/tc-redirect-tap

docs:
	@echo ">>>> Build docs"
	$(MAKE) -C $@

all: build
