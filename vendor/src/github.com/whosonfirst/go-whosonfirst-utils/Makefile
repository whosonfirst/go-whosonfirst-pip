prep:
	if test -d pkg; then rm -rf pkg; fi

self:   prep
	if test -d src/github.com/whosonfirst/go-whosonfirst-utils; then rm -rf src/github.com/whosonfirst/go-whosonfirst-utils; fi
	mkdir -p src/github.com/whosonfirst/go-whosonfirst-utils
	cp utils.go src/github.com/whosonfirst/go-whosonfirst-utils/utils.go

deps:	self

rmdeps:
	if test -d src; then rm -rf src; fi 

build:	rmdeps deps fmt bin

fmt:	self
	go fmt utils.go
	go fmt cmd/*.go

expand: self
	go build -o bin/wof-expand cmd/wof-expand.go

bin:
	@GOPATH=$(shell pwd) go build -o bin/wof-expand cmd/wof-expand.go
	@GOPATH=$(shell pwd) go build -o bin/wof-hash cmd/wof-hash.go
