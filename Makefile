#Dockerfile vars

#vars
TAG=v0.2.0
UID=`id -u`
GID=`id -g`
BUILDDATE=${shell date -u +%Y-%m-%dT%H:%M:%SZ}
IMAGENAME=mesos-firecracker-executor
IMAGEFULLNAME=avhost/${IMAGENAME}
BRANCH=${TAG}
BRANCHSHORT=$(shell echo ${BRANCH} | awk -F. '{ print $$1"."$$2 }')
LASTCOMMIT=$(shell git log -1 --pretty=short | tail -n 1 | tr -d " " | tr -d "UPDATE:")
ARCH=$(shell uname -m)


# Version has to be in format of vx.x.x
firecracker_version=v1.13.1
release_url=https://github.com/firecracker-microvm/firecracker/releases/download/$(firecracker_version)/firecracker-$(firecracker_version)-$(ARCH).tgz

# --location is needed to follow redirects
curl = curl --location

.PHONY: help build all docs tc-redirect-tap firecracker

.DEFAULT_GOAL := all

define install_go
GOBIN=$(abspath resources) go install $(1)@$(2)
endef


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

tc-redirect-tap:
	$(call install_go,github.com/awslabs/tc-redirect-tap/cmd/tc-redirect-tap,latest)

firecracker:
	$(curl) ${release_url} | tar -xvzf - -C /tmp
	mv /tmp/release-$(firecracker_version)-$(ARCH)/firecracker-$(firecracker_version)-$(ARCH) resources/firecracker/firecracker
	mv /tmp/release-$(firecracker_version)-$(ARCH)/jailer-$(firecracker_version)-$(ARCH) resources/firecracker/jailer
	mv /tmp/release-$(firecracker_version)-$(ARCH)/rebase-snap-$(firecracker_version)-$(ARCH) resources/firecracker/rebase-snap
	mv /tmp/release-$(firecracker_version)-$(ARCH)/seccompiler-bin-$(firecracker_version)-$(ARCH) resources/firecracker/seccompiler-bin
	mv /tmp/release-$(firecracker_version)-$(ARCH)/seccomp-filter-$(firecracker_version)-$(ARCH).json resources/firecracker/seccomp-filter.json
	rm -rf /tmp/release-$(firecracker_version)-$(ARCH)

check: go-fmt sboom seccheck imagecheck
all: submodule-update check tc-redirect firecracker build
vmm: build-vmm 
