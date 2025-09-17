#Dockerfile vars

#vars
TAG=v0.1.0
UID=`id -u`
GID=`id -g`
BUILDDATE=${shell date -u +%Y-%m-%dT%H:%M:%SZ}
IMAGENAME=mesos-firecracker-executor
IMAGEFULLNAME=avhost/${IMAGENAME}
BRANCH=${TAG}
BRANCHSHORT=$(shell echo ${BRANCH} | awk -F. '{ print $$1"."$$2 }')
LASTCOMMIT=$(shell git log -1 --pretty=short | tail -n 1 | tr -d " " | tr -d "UPDATE:")

.PHONY: help build all docs 

.DEFAULT_GOAL := all


build-bin:
	@echo ">>>> Build binary"
	@CGO_ENABLED=0 GOOS=linux go build  -ldflags "-s -w -X main.BuildVersion=${BUILDDATE} -X main.GitVersion=${TAG}  -extldflags \"-static\"" -o build/mesos-firecracker-executor .

build: build-bin
	@echo ">>>> Build docker image branch: latest"
	@docker build -t ${IMAGEFULLNAME}:latest -f Dockerfile .

submodule-update:
	@git pull --recurse-submodules
	@git submodule update --recursive --remote

build-vmm: 
	@cd resources/vmm-agent/; ${MAKE}
	@cp resources/vmm-agent/build/vmm-agent build/
	@cp resources/vmm-agent/build/vmlinux build/


update-gomod:
	go get -u
	go mod tidy

seccheck:
	grype --add-cpes-if-none .

sboom:
	syft dir:. > sbom.txt
	syft dir:. -o json > sbom.json

imagecheck:
	grype --add-cpes-if-none ${IMAGEFULLNAME}:latest > cve-report.md

go-fmt:
	@gofmt -w .

push:
	@echo ">>>> Publish docker image: " ${BRANCH} ${BRANCHSHORT}
	-docker buildx create --use --name buildkit
	@docker buildx build --sbom=true --provenance=true --platform linux/amd64 --push --build-arg TAG=${TAG} --build-arg BUILDDATE=${BUILDDATE} -t ${IMAGEFULLNAME}:${BRANCH} .
	@docker buildx build --sbom=true --provenance=true --platform linux/amd64 --push --build-arg TAG=${TAG} --build-arg BUILDDATE=${BUILDDATE} -t ${IMAGEFULLNAME}:${BRANCHSHORT} .
	@docker buildx build --sbom=true --provenance=true --platform linux/amd64 --push --build-arg TAG=${TAG} --build-arg BUILDDATE=${BUILDDATE} -t ${IMAGEFULLNAME}:latest .
	-docker buildx rm buildkit


check: go-fmt sboom seccheck imagecheck
all: check submodule-update check build
vmm: build-vmm 
