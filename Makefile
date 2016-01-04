prep:
	if test -d pkg; then rm -rf pkg; fi

self:   prep
	if test -d src/github.com/whosonfirst/go-whosonfirst-pip; then rm -rf src/github.com/whosonfirst/go-whosonfirst-pip; fi
	mkdir -p src/github.com/whosonfirst/go-whosonfirst-pip
	cp pip.go src/github.com/whosonfirst/go-whosonfirst-pip/

deps:   self
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-geojson"
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-utils"
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-csv"
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-log"
	@GOPATH=$(shell pwd) go get -u "github.com/dhconnelly/rtreego"
	@GOPATH=$(shell pwd) go get -u "github.com/hashicorp/golang-lru"
	@GOPATH=$(shell pwd) go get -u "github.com/rcrowley/go-metrics"

fmt:
	go fmt cmd/*.go
	go fmt *.go

bin: 	self
	@GOPATH=$(shell pwd) go build -o bin/wof-pip-index cmd/wof-pip-index.go
	@GOPATH=$(shell pwd) go build -o bin/wof-pip-index-csv cmd/wof-pip-index-csv.go
	@GOPATH=$(shell pwd) go build -o bin/wof-pip-server cmd/wof-pip-server.go
	@GOPATH=$(shell pwd) go build -o bin/wof-pip-proxy cmd/wof-pip-proxy.go
