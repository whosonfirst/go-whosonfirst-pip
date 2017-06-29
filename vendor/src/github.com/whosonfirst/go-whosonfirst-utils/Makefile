CWD=$(shell pwd)
GOPATH := $(CWD)

prep:
	if test -d pkg; then rm -rf pkg; fi

self:   prep
	if test -d src/github.com/whosonfirst/go-whosonfirst-utils; then rm -rf src/github.com/whosonfirst/go-whosonfirst-utils; fi
	mkdir -p src/github.com/whosonfirst/go-whosonfirst-utils
	cp utils.go src/github.com/whosonfirst/go-whosonfirst-utils/utils.go
	cp -r vendor/src/* src/

deps:	
	@GOPATH=$(GOPATH) go get -u "github.com/tidwall/gjson"
	@GOPATH=$(GOPATH) go get -u "github.com/whosonfirst/go-whosonfirst-crawl"
	@GOPATH=$(GOPATH) go get -u "github.com/whosonfirst/go-whosonfirst-csv"
	@GOPATH=$(GOPATH) go get -u "github.com/whosonfirst/go-whosonfirst-uri"

rmdeps:
	if test -d src; then rm -rf src; fi 

build:	rmdeps deps fmt bin

vendor-deps: rmdeps deps
	if test ! -d vendor; then mkdir vendor; fi
	if test -d vendor/src; then rm -rf vendor/src; fi
	cp -r src vendor/src
	find vendor -name '.git' -print -type d -exec rm -rf {} +
	rm -rf src

fmt:	self
	go fmt utils.go
	go fmt cmd/*.go

bin:	self
	@GOPATH=$(shell pwd) go build -o bin/wof-cat cmd/wof-cat.go
	@GOPATH=$(shell pwd) go build -o bin/wof-compare cmd/wof-compare.go
	@GOPATH=$(shell pwd) go build -o bin/wof-d2fc cmd/wof-d2fc.go
	@GOPATH=$(shell pwd) go build -o bin/wof-geojsonls-dump cmd/wof-geojsonls-dump.go
	@GOPATH=$(shell pwd) go build -o bin/wof-geojsonls-validate cmd/wof-geojsonls-validate.go
	@GOPATH=$(shell pwd) go build -o bin/wof-ensure-property cmd/wof-ensure-property.go
	@GOPATH=$(shell pwd) go build -o bin/wof-expand cmd/wof-expand.go
	@GOPATH=$(shell pwd) go build -o bin/wof-hash cmd/wof-hash.go
