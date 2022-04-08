#Dockerfile vars

#vars
TAG=`git describe`
BUILDDATE=`date -u +%Y-%m-%dT%H:%M:%SZ`
BRANCH=`git symbolic-ref --short HEAD`

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

docs:
	@echo ">>>> Build docs"
	$(MAKE) -C $@

all: build 
