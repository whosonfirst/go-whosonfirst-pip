# go-whosonfirst-pip

Expermimental point-in-polygon library for Who's On First documents

## Set up

### Setting up your Go path

```
export GOPATH=`pwd`
```

_Adjust accordingly if you are not use a different shell than Bash._

### Dependencies

```
go get -u "github.com/whosonfirst/go-whosonfirst-geojson"
go get -u "github.com/whosonfirst/go-whosonfirst-utils"
go get -u "github.com/kellydunn/golang-geo"
go get -u "github.com/dhconnelly/rtreego"
```

There is also a helpful `deps` target in the included Makefile to do this for you.

### Mirror mirror on the wall

```
if test -d pkg; then rm -rf pkg; fi
if test -d src/github.com/whosonfirst/go-whosonfirst-pip; then rm -rf src/github.com/whosonfirst/go-whosonfirst-pip; fi
mkdir -p src/github.com/whosonfirst/go-whosonfirst-pip
cp *.go src/github.com/whosonfirst/go-whosonfirst-pip/
```

I am still a bit lost and baffled by how and where Go thinks to look for stuff. In order to make standalone tools compile locally I just clone any package specific code in to the `src` directory. It's not pretty but it works.

There is also a helpful `self` target in the included Makefile to do this for you.

### Tools

```
go build -o bin/index bin/index.go
go build -o bin/index-csv bin/index-csv.go
go build -o bin/pip-server bin/pip-server.go
```

There is also a helpful `bin` target in the included Makefile to do this for you.

## Usage

### The basics

```
package main

import (
	"github.com/whosonfirst/go-whosonfirst-pip"
)

source := "/usr/local/mapzen/whosonfirst-data"
p := pip.PointInPolygon(source)

geojson_file := "/usr/local/mapzen/whosonfirst-data/data/101/736/545/101736545.geojson"
p.IndexGeoJSONFile(geojson_file)

# Or this:

meta_file := "/usr/local/mapzen/whosonfirst-data/meta/wof-locality-latest.csv"
p.IndexMetaJSONFile(meta_file)
```

You can index individual GeoJSON files or [Who's On First "meta" files](https://github.com/whosonfirst/whosonfirst-data/tree/master/meta) which are just CSV files with pointers to individual Who's On First records.

The `PointInPolygon` function takes as its sole argument the root path where your Who's On First documents are stored. This is because those files are used to perform a final "containment" check. The details of this are discussed further below.

### Simple

```

lat := 40.677524
lon := -73.987343

results, timings := p.GetByLatLon(lat, lon)

for i, f := range results {
	fmt.Printf("simple result #%d is %s\n", i, f.Name)
}

for _, t := range timings {
        fmt.Printf("[timing] %s: %f\n", t.Event, t.Duration)
}
```

`results` contains a list of `geojson.WOFSpatial` object-interface-struct-things and `timings` contains a list of `pip.WOFPointInPolygonTiming` object-interface-struct-things. 

### What's going on under the hood

```
results := p.GetIntersectsByLatLon(lat, lon)
inflated := p.InflateSpatialResults(results)

for i, wof := range inflated {
	fmt.Printf("result #%d is %s\n", i, wof.Name)
}

filtered := p.FilterByPlacetype(inflated, "locality")

for i, f := range filtered {
	fmt.Printf("filtered result #%d is %s\n", i, f.Name)
}

contained := p.EnsureContained(lat, lon, inflated)

for i, f := range contained {
	fmt.Printf("contained result #%d is %s\n", i, f.Name)
}

```

If you're curious how the sausage is made.

### HTTP Ponies

There is also a standalone HTTP server for performing point-in-polygon lookups. It is instantiated with a `source` parameter and one or more "meta" CSV files, like this:

```
$> ./bin/pip-server -source /usr/local/mapzen/whosonfirst-data/data /usr/local/mapzen/whosonfirst-data/meta/wof-neighbourhood-latest.csv
indexed 49906 records in 47.895 seconds
```

This is how you'd use it:

```
$> curl 'http://localhost:8080?latitude=40.677524&longitude=-73.987343' | python -mjson.tool
[
    {
        "Id": 102061079,
        "Name": "Gowanus Heights",
        "Placetype": "neighbourhood"
    },
    {
        "Id": 85865587,
        "Name": "Gowanus",
        "Placetype": "neighbourhood"
    }
]
```

## Assumptions, caveats and known-knowns

### When we say `geojson` in the context of Go-typing

We are talking about the [go-whosonfirst-geojson](https://www.github.com/whosonfirst/go-whosonfirst-geojson) library

### Speed and performance

This is how it works now:

1. We are using the [rtreego](https://www.github.com/dhconnelly/rtreego) library to do most of the heavy lifting and filtering
2. Results from the rtreego `SearchIntersect` method are "inflated" and recast as geojson `WOFSpatial` object-interface-struct-things
3. We are performing a final containment check on the results by reading the corresponding GeoJSON file and reading its geometry in to one or more [golang-geo](https://www.github.com/kellydunn/golang-geo) `Polygon` object-interface-struct-things. Each of these object-interface-struct-things calls its `Contains` method on an input coordinate.

This is how long it takes reverse-geocoding a point in Brooklyn, using an index of all the countries in Who's On First:

```
[timings] 40.677524,-73.987343 ()
[timing] intersects: 0.000030
[timing] inflate: 0.000000
[timing] contained: 0.115600
```

These numbers are still a bit vague and misleading. For example it's not clear (because it hasn't been measured yet) where most of the work in that 0.1 seconds is happening. Is it reading the GeoJSON file? It is converting the file's geometry in to Polygon object-interface-struct-things? It is actually testing a single coordinate against a giant bag of coordinates? I don't know, yet.

Whatever the case there is lots of room for making this "more fast".

## See also

* https://www.github.com/dhconnelly/rtreego
* https://www.github.com/kellydunn/golang-geo
* https://www.github.com/whosonfirst/go-whosonfirst-geojson
* https://whosonfirst.mapzen.com/data/