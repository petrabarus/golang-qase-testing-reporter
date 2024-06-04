# Check for required command tools to build or stop immediately

ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

BINARY=go-qase-testing-reporter
VERSION=1.0.0
BUILD=`git rev-parse HEAD`
BUILD_DIR=build
PLATFORMS=darwin linux windows
ARCHITECTURES=386 amd64

# Setup linker flags option for build that interoperate with variable names in src code
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.Build=${BUILD}"

.PHONY: check clean install build_all all

default: build

all: clean build_all install

build:
	mkdir -p ${BUILD_DIR}
	go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY} .

build_all:
	mkdir -p ${BUILD_DIR}
	$(foreach GOOS, $(PLATFORMS),\
	$(foreach GOARCH, $(ARCHITECTURES), $(shell export GOOS=$(GOOS); export GOARCH=$(GOARCH); go build ${LDFLAGS} -v -o ${BUILD_DIR}/$(BINARY)-${VERSION}-$(GOOS)-$(GOARCH))))

install:
	go install ${LDFLAGS}

clean:
	rm -rf ${BUILD_DIR}

generate_api:
	oapi-codegen -generate types,client -package main -o api.gen.go api.yaml