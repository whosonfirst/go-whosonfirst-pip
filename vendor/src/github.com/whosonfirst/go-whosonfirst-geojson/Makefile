prep:
	if test -d pkg; then rm -rf pkg; fi

self:   prep
	if test -d src/github.com/whosonfirst/go-whosonfirst-geojson; then rm -rf src/github.com/whosonfirst/go-whosonfirst-geojson; fi
	mkdir -p src/github.com/whosonfirst/go-whosonfirst-geojson
	cp geojson.go src/github.com/whosonfirst/go-whosonfirst-geojson/geojson.go

deps:   self
	go get -u "github.com/jeffail/gabs"
	go get -u "github.com/dhconnelly/rtreego"
	go get -u "github.com/kellydunn/golang-geo"
	go get -u "github.com/whosonfirst/go-whosonfirst-crawl"

fmt:
	go fmt cmd/*.go
	go fmt *.go

bin:	self
	go build -o bin/wof-geojson-contains cmd/wof-geojson-contains.go
	go build -o bin/wof-geojson-dump cmd/wof-geojson-dump.go
	go build -o bin/wof-geojson-enspatialize cmd/wof-geojson-enspatialize.go
	go build -o bin/wof-geojson-polygons cmd/wof-geojson-polygons.go
	go build -o bin/wof-geojson-validate cmd/wof-geojson-validate.go
