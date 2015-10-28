prep:
	if test -d pkg; then rm -rf pkg; fi

self:   prep
	if test -d src/github.com/whosonfirst/go-whosonfirst-pip; then rm -rf src/github.com/whosonfirst/go-whosonfirst-pip; fi
	mkdir -p src/github.com/whosonfirst/go-whosonfirst-pip
	cp pip.go src/github.com/whosonfirst/go-whosonfirst-pip/

deps:   self
	go get -u "github.com/whosonfirst/go-whosonfirst-geojson"
	go get -u "github.com/whosonfirst/go-whosonfirst-utils"
	go get -u "github.com/whosonfirst/go-whosonfirst-csv"
	go get -u "github.com/whosonfirst/go-whosonfirst-log"
	go get -u "github.com/dhconnelly/rtreego"
	go get -u "github.com/hashicorp/golang-lru"
	go get -u "github.com/rcrowley/go-metrics"

fmt:
	go fmt bin/*.go
	go fmt *.go

bin: 	self
	go build -o bin/index bin/index.go
	go build -o bin/index-csv bin/index-csv.go
	go build -o bin/pip-server bin/pip-server.go
