.PHONY: build image clean

TARGET ?= lxcfs-sidecar-injector
ODIR ?= _output
IMG ?= crazytaxii/lxcfs-sidecar-injector
TAG ?= latest
GOOS ?= linux
GOARCH ?= amd64
CGO_ENABLED ?= 0
REPO_URL ?= $(shell git remote get-url origin)
BRANCH ?= $(shell git branch --show-current)
COMMIT_REF ?= $(shell git rev-parse --verify HEAD)

build:
	go build -o $(ODIR)/$(TARGET) ./cmd/injector

image:
	docker build --build-arg REPO_URL=$(REPO_URL) --build-arg BRANCH=$(BRANCH) --build-arg COMMIT_REF=$(COMMIT_REF) -t $(IMG):$(TAG) .

push:
	docker push $(IMG):$(TAG)

clean:
	# Remove binary built
	rm -rf $(ODIR)/$(TARGET)
