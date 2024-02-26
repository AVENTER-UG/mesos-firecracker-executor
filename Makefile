#Dockerfile vars

#vars
TAG=dev
UID=`id -u`
GID=`id -g`
TAG=${shell git describe}
BUILDDATE=${shell date -u +%Y-%m-%dT%H:%M:%SZ}
IMAGENAME=mesos-firecracker-executor
IMAGEFULLNAME=avhost/${IMAGENAME}
BRANCH=`git symbolic-ref --short HEAD`
LASTCOMMIT=$(shell git log -1 --pretty=short | tail -n 1 | tr -d " " | tr -d "UPDATE:")

.PHONY: help build all docs 

.DEFAULT_GOAL := all

ifeq (${BRANCH}, master) 
        BRANCH=latest
endif

ifneq ($(shell echo $(LASTCOMMIT) | grep -E '^v([0-9]+\.){0,2}(\*|[0-9]+)'),)
        BRANCH=${LASTCOMMIT}
else
        BRANCH=latest
endif

build-bin:
	@echo ">>>> Build binary"
	@CGO_ENABLED=0 GOOS=linux go build  -ldflags "-s -w -X main.BuildVersion=${BUILDDATE} -X main.GitVersion=${TAG}  -extldflags \"-static\"" -o build/mesos-firecracker-executor .

build: build-bin
	@echo ">>>> Build docker image branch: latest"
	@docker build -t avhost/mesos-firecracker-executor:latest -f Dockerfile .

submodule-update:
	@git pull --recurse-submodules
	@git submodule update --recursive --remote

build-vmm: 
	@cd resources/vmm-agent/; ${MAKE}
	@cp resources/vmm-agent/build/vmm-agent build/
	@cp resources/vmm-agent/build/vmlinux build/

seccheck:
	grype --add-cpes-if-none .

sboom:
	syft dir:. > sbom.txt
	syft dir:. -o json > sbom.json

go-fmt:
	@gofmt -w .

update-gomod:
	go get -u

push:
	@echo ">>>> Publish docker image: " ${BRANCH}
	@docker buildx build --platform linux/amd64 --push --build-arg TAG=${TAG} --build-arg BUILDDATE=${BUILDDATE} -t ${IMAGEFULLNAME}:latest .
	@docker buildx build --platform linux/amd64 --push --build-arg TAG=${TAG} --build-arg BUILDDATE=${BUILDDATE} -t ${IMAGEFULLNAME}:${BRANCH} .

check: go-fmt sboom seccheck
all: submodule-update check build-vmm build
vmm: build-vmm 
