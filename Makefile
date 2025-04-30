DOCKER_REPO=registry.blackforestbytes.com
DOCKER_NAME=mikescher/simple-stupid-proxy

NAMESPACE=$(shell git rev-parse --abbrev-ref HEAD)
HASH=$(shell git rev-parse HEAD)

.PHONY: run build fmt dgi test lint clean swagger docker push run-docker-local inspect-docker

run: build
	mkdir -p _run-data
	_build/server

build: fmt
	mkdir -p _build
	rm -f ./_build/server
	go build -v -buildvcs=false  -o _build/server ./cmd/server

fmt:
	go fmt ./...

clean:
	rm -rf _build/*
	git clean -fdx --exclude=./config
	! which go 2>&1 >> /dev/null || go clean
	! which go 2>&1 >> /dev/null || go clean -testcache

push-bfb:
	docker image push "$(DOCKER_REPO)/$(DOCKER_NAME):$(HASH)"
	docker image push "$(DOCKER_REPO)/$(DOCKER_NAME):$(NAMESPACE)-latest"
	docker image push "$(DOCKER_REPO)/$(DOCKER_NAME):latest"

push-pub:
	docker image push "$(DOCKER_NAME):$(HASH)"
	docker image push "$(DOCKER_NAME):$(NAMESPACE)-latest"
	docker image push "$(DOCKER_NAME):latest"

dgi:
	[ ! -f "DOCKER_GIT_INFO" ] || rm DOCKER_GIT_INFO
	echo -n "VCSTYPE="     >> DOCKER_GIT_INFO ; echo "git"                         >> DOCKER_GIT_INFO
	echo -n "BRANCH="      >> DOCKER_GIT_INFO ; git rev-parse --abbrev-ref HEAD    >> DOCKER_GIT_INFO
	echo -n "HASH="        >> DOCKER_GIT_INFO ; git rev-parse              HEAD    >> DOCKER_GIT_INFO
	echo -n "COMMITTIME="  >> DOCKER_GIT_INFO ; git log -1 --format=%cd --date=iso >> DOCKER_GIT_INFO
	echo -n "REMOTE="      >> DOCKER_GIT_INFO ; git config --get remote.origin.url >> DOCKER_GIT_INFO

docker: dgi
	docker build \
    		-t "$(DOCKER_NAME):$(HASH)" \
    		-t "$(DOCKER_NAME):$(NAMESPACE)-latest" \
    		-t "$(DOCKER_NAME):latest" \
    		-t "$(DOCKER_REPO)/$(DOCKER_NAME):$(HASH)" \
    		-t "$(DOCKER_REPO)/$(DOCKER_NAME):$(NAMESPACE)-latest" \
    		-t "$(DOCKER_REPO)/$(DOCKER_NAME):latest" \
    		.

run-interactive:
	docker build -t $(DOCKER_NAME)/local -f DockerfileLocal .
	docker run -d --rm -p 8080:80 -v $(shell pwd):/src $(DOCKER_NAME)/local

run-docker-local: docker
	mkdir -p _run-data
	docker run --rm \
	           --init \
			   --volume "$(shell pwd)/_run-data/docker-local:/data" \
			   --publish "8080:80" \
			   $(DOCKER_NAME):latest

inspect-docker: docker
	mkdir -p _run-data
	docker run -ti \
	           --rm \
	           --volume "$(shell pwd)/_run-data/docker-inspect:/data" \
	           $(DOCKER_NAME):latest \
	           bash
