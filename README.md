# go-whosonfirst-pip

Expermimental in-memory point-in-polygon (reverse geocoding) library for Who's On First data

## Set up

### Setting up your Go path

```
export GOPATH=`pwd`
```

_Adjust accordingly if you are using a shell other than Bash._

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
p.IndexMetaFile(meta_file)
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

for i, r := range results {
	fmt.Printf("spatial result #%d is %v\n", i, r)
}

inflated := p.InflateSpatialResults(results)

for i, wof := range inflated {
	fmt.Printf("wof result #%d is %s\n", i, wof.Name)
}

# This only happens when you call `GetByLatLonForPlacetype`

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

We are talking about the [go-whosonfirst-geojson](https://www.github.com/whosonfirst/go-whosonfirst-geojson) library.

### Speed and performance

This is how it works now:

1. We are using the [rtreego](https://www.github.com/dhconnelly/rtreego) library to do most of the heavy lifting and filtering.
2. Results from the rtreego `SearchIntersect` method are "inflated" and recast as geojson `WOFSpatial` object-interface-struct-things.
3. We are performing a final containment check on the results by reading each corresponding GeoJSON file and converting its geometry in to one or more [golang-geo](https://www.github.com/kellydunn/golang-geo) `Polygon` object-interface-struct-things. Each of these object-interface-struct-things calls its `Contains` method on an input coordinate.

This is how long it takes reverse-geocoding a point in Brooklyn, using an index of all the countries in Who's On First:

```
[timings] 40.677524,-73.987343 ()
[timing] intersects: 0.000030
[timing] inflate: 0.000000
[timing] contained: 0.115600
```

If we break that down a bit more we can see that most of the time is spent reading/parsing (does it matter?) the GeoJSON files from disk:

```
./bin/pip-server -source /usr/local/mapzen/whosonfirst-data/data /usr/local/mapzen/whosonfirst-data/meta/wof-neighbourhood-latest.csv /usr/local/mapzen/whosonfirst-data/meta/wof-country-latest.csv 
indexed 50124 records in 60.472 seconds 
time to unmarshal /usr/local/mapzen/whosonfirst-data/data/102/061/079/102061079.geojson is 0.000108
time to convert geom to polygons is 0.000009
time to check containment (true) after 1/1 possible iterations is 0.000002
time to unmarshal /usr/local/mapzen/whosonfirst-data/data/856/337/93/85633793.geojson is 0.103965
time to convert geom to polygons is 0.010570
time to check containment (true) after 75/75 possible iterations is 0.000935
time to unmarshal /usr/local/mapzen/whosonfirst-data/data/858/655/87/85865587.geojson is 0.000300
time to convert geom to polygons is 0.000044
time to check containment (true) after 2/2 possible iterations is 0.000002
time to unmarshal /usr/local/mapzen/whosonfirst-data/data/858/406/09/85840609.geojson is 0.000248
time to convert geom to polygons is 0.000017
time to check containment (false) after 1/1 possible iterations is 0.000001
contained: 3/4
[timings] 40.677524, -73.987343 (3 results)
[timing] intersects: 0.000236
[timing] inflate: 0.000001
[timing] contained: 0.116374
```

So, that's a known-known. On the other hand unless you're doing a lot of reverse-geocoding around convergent international borders it's probably not going to be that big a deal. For example:

```
$> siege -c 100 -i -f urls2.txt 
** SIEGE 3.0.5
** Preparing 100 concurrent users for battle.
The server is now under siege...^C
Lifting the server siege...      done.

Transactions:			136924 hits
Availability:			100.00 %
Elapsed time:			756.74 secs
Data transferred:		4.79 MB
Response time:			0.05 secs
Transaction rate:		180.94 trans/sec
Throughput:			0.01 MB/sec
Concurrency:			9.92
Successful transactions:	136924
Failed transactions:		0
Longest transaction:		0.79
Shortest transaction:		0.00

```

Or maybe files with (n) number of polygons / coordinates could be cached in memory. Or something. Whatever the case there is room for making this "Moar Faster".

_If you're wondering, sorting the polygons by largest number of coordinates before iterating over them doesn't appear to have any meaningful performance improvement._

### Using this with other data sources

Yeah... _probably_. Not? _Yet._

There's nothing in this library per se that would prevent you from using it with any old bag of GeoJSON. It's more that this library uses [go-whosonfirst-geojson](https://www.github.com/whosonfirst/go-whosonfirst-geojson) which _does_ make some WOF-specific assumptions.

Specifically, in the `EnSpatialize` method which is used to generate a `rtreego.Spatial` compatible interface, like this:

```
func (wof WOFFeature) EnSpatialize() (*WOFSpatial, error) {

     // These all look for things prefixed by "wof:" in the properties hash

     id := wof.WOFId()
     name := wof.WOFName()
     placetype := wof.WOFPlacetype()
```

So that should be changed, or made WOF-specific. Or something. Because yes, you ought to be able to use this (and the `go-whosonfirst-geojson`) library with any old GeoJSON file out there. But not today.

### Less-than-perfect GeoJSON files

First, these should not be confused with malformed GeoJSON files. Some records in Who's On First are missing geometries or maybe the geometries are... well, less than perfect. The `rtreego` package is very strict about what it can handle and freaks out and dies rather than returning errors. So that's still a thing. Personally I like the idea of using `pip-server` as a kind of unfriendly validation tool for Who's On First data but it also means that, for the time being, it is understood that some records may break everything.

## See also

* https://www.github.com/dhconnelly/rtreego
* https://www.github.com/kellydunn/golang-geo
* https://www.github.com/whosonfirst/go-whosonfirst-geojson
* https://whosonfirst.mapzen.com/data/