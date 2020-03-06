-include .env

VERSION := $(shell git describe --tags 2>/dev/null || echo 0.1)
BUILD := $(shell git rev-parse --short HEAD)
PROJECTNAME := $(shell basename "$(PWD)")

# Go related variables.
GOBASE := $(shell pwd)
GOPATH := $(GOBASE)/vendor:$(GOBASE):$(GOPATH)
GOBIN := $(GOBASE)/build

# Use linker flags to provide version/build settings
LDFLAGS=-ldflags "-X=main.Version=$(VERSION) -X=main.Build=$(BUILD)"

# Redirect error output to a file, so we can show it in development mode.
STDERR := /tmp/.$(PROJECTNAME)-stderr.txt

# PID file will keep the process id of the server
PID := /tmp/.$(PROJECTNAME).pid

# Make is verbose in Linux. Make it silent.
MAKEFLAGS += --silent

clean:
	@echo "  >  Cleaning build cache"
	@-rm $(GOBIN)/$(PROJECTNAME) 2> /dev/null
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go clean

build:
	echo "Building $(PROJECTNAME)"
	GOPATH=$(GOPATH) GOBIN=$(GOBIN) go build $(LDFLAGS) -o $(GOBIN)/$(PROJECTNAME)

run-server: build
	sudo $(GOBIN)/$(PROJECTNAME)

.PHONY: help modules-build
all: help
help: Makefile
	@echo
	@echo " Choose a command run in "$(PROJECTNAME)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo

