CWD=$(shell pwd)
GOPATH := $(CWD)

prep:
	if test -d pkg; then rm -rf pkg; fi

self:   prep rmdeps
	if test -d src/github.com/whosonfirst/go-whosonfirst-pip; then rm -rf src/github.com/whosonfirst/go-whosonfirst-pip; fi
	mkdir -p src/github.com/whosonfirst/go-whosonfirst-pip
	cp pip.go src/github.com/whosonfirst/go-whosonfirst-pip/
	cp -r vendor/src/* src/

rmdeps:
	if test -d src; then rm -rf src; fi 

build:	fmt bin

deps:	rmdeps
	@GOPATH=$(GOPATH) go get -u "github.com/whosonfirst/go-whosonfirst-geojson"
	@GOPATH=$(GOPATH) go get -u "github.com/whosonfirst/go-whosonfirst-utils"
	@GOPATH=$(GOPATH) go get -u "github.com/whosonfirst/go-whosonfirst-csv"
	@GOPATH=$(GOPATH) go get -u "github.com/whosonfirst/go-whosonfirst-log"
	@GOPATH=$(GOPATH) go get -u "github.com/dhconnelly/rtreego"
	@GOPATH=$(GOPATH) go get -u "github.com/hashicorp/golang-lru"
	@GOPATH=$(GOPATH) go get -u "github.com/rcrowley/go-metrics"
	@GOPATH=$(GOPATH) go get -u "github.com/facebookgo/grace/gracehttp"

vendor: deps
	if test ! -d vendor; then mkdir vendor; fi
	if test -d vendor/src; then rm -rf vendor/src; fi
	cp -r src vendor/src
	find vendor -name '.git' -print -type d -exec rm -rf {} +
	rm -rf src

fmt:
	go fmt cmd/*.go
	go fmt *.go

bin: 	self
	@GOPATH=$(GOPATH) go build -o bin/wof-pip-index cmd/wof-pip-index.go
	@GOPATH=$(GOPATH) go build -o bin/wof-pip-index-csv cmd/wof-pip-index-csv.go
	@GOPATH=$(GOPATH) go build -o bin/wof-pip-server cmd/wof-pip-server.go
	@GOPATH=$(GOPATH) go build -o bin/wof-pip-proxy cmd/wof-pip-proxy.go
