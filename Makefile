.PHONY: build clean

TARGET ?= lxcfs-sidecar-injector
ODIR ?= _output
IMG ?= lxcfs-sidecar-injector
TAG ?= latest
GOOS ?= linux
GOARCH ?= amd64
CGO_ENABLED ?= 0

build:
	go build -o $(ODIR)/$(TARGET) ./cmd/injector

image:
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) go build -o $(ODIR)/$(TARGET) ./cmd/injector
	docker build -t $(IMG):$(TAG) .

clean:
	# Remove binary built
	rm -rf $(ODIR)/$(TARGET)
