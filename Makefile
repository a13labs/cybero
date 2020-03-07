-include .env

VERSION := $(shell git describe --tags 2>/dev/null || echo 0.1)
BUILD := $(shell git rev-parse --short HEAD)
PROJECTNAME := $(shell basename "$(PWD)")
APPSROOT := apps
APPS := $(foreach APP,$(APPSROOT),$(wildcard $(APP)/*))
APPSGOFILES := $(foreach APPDIR,$(APPS),$(wildcard $(APPDIR)/*.go)) 

MODULESROOT := modules

# Server Modules
SRVMODULESFAMILY := $(foreach SRVMODFAMILY,$(MODULESROOT),$(wildcard $(SRVMODFAMILY)/server/*))
SRVMODULES := $(foreach SRVMODULE,$(SRVMODULESFAMILY),$(wildcard $(SRVMODULE)/*))
SRVMODULESGOFILES := $(foreach SRVMODULEDIR,$(SRVMODULES),$(wildcard $(SRVMODULEDIR)/*.go)) 

# Client Modules
CLIMODULESFAMILY := $(foreach CLIMODFAMILY,$(MODULESROOT),$(wildcard $(CLIMODFAMILY)/server/*))
CLIMODULES := $(foreach CLIMODULE,$(CLIMODULESFAMILY),$(wildcard $(CLIMODULE)/*))
CLIMODULESGOFILES := $(foreach CLIMODULEDIR,$(CLIMODULES),$(wildcard $(CLIMODULEDIR)/*.go)) 

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

## install: Install missing dependencies. Runs `go get` internally. e.g; make install get=github.com/foo/bar
# install: go-get

# ## start: Start in development mode. Auto-starts when code changes.
# start:
# 	@bash -c "trap 'make stop' EXIT; $(MAKE) clean compile start-server watch run='make clean compile start-server'"

# ## stop: Stop development mode.
# stop: stop-server

# start-server: stop-server
# 	@echo "  >  $(PROJECTNAME) is available at $(ADDR)"
# 	@-$(GOBIN)/$(PROJECTNAME) 2>&1 & echo $$! > $(PID)
# 	@cat $(PID) | sed "/^/s/^/  \>  PID: /"

# stop-server:
# 	@-touch $(PID)
# 	@-kill `cat $(PID)` 2> /dev/null || true
# 	@-rm $(PID)

# ## watch: Run given command when code changes. e.g; make watch run="echo 'hey'"
# watch:
# 	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) yolo -i . -e vendor -e bin -c "$(run)"

# restart-server: stop-server start-server

# ## compile: Compile the binary.
# compile:
# 	@-touch $(STDERR)
# 	@-rm $(STDERR)
# 	@-$(MAKE) -s go-compile 2> $(STDERR)
# 	@cat $(STDERR) | sed -e '1s/.*/\nError:\n/'  | sed 's/make\[.*/ /' | sed "/^/s/^/     /" 1>&2

# ## exec: Run given command, wrapped with custom GOPATH. e.g; make exec run="go test ./..."
# exec:
# 	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) $(run)

# ## clean: Clean build files. Runs `go clean` internally.
# clean:
# 	@-rm $(GOBIN)/$(PROJECTNAME) 2> /dev/null
# 	@-$(MAKE) go-clean

# go-install:
# 	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go install $(GOFILES)

clean:
	@echo "  >  Cleaning build cache"
	@-rm $(GOBIN)/$(PROJECTNAME) 2> /dev/null
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go clean

server-modules-build: $(SRVMODULESGOFILES)

apps-build: $(APPSGOFILES)

$(SRVMODULESGOFILES): %.go
	echo "Building $(basename $@)"
	GOPATH=$(GOPATH) GOBIN=$(GOBIN) go build -buildmode=plugin $(LDFLAGS) -o $(GOBIN)/$(PROJECTNAME)/modules/server/$(basename $(notdir $@)).so $@

$(APPSGOFILES): %.go
	echo "Building $(basename $@)"
	GOPATH=$(GOPATH) GOBIN=$(GOBIN) go build $(LDFLAGS) -o $(GOBIN)/$(PROJECTNAME)/$(basename $(notdir $@)) $@

build: server-modules-build apps-build

%.go: 
	@echo "Building...." 

.PHONY: help server-modules-build
all: help
help: Makefile
	@echo
	@echo " Choose a command run in "$(PROJECTNAME)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo

