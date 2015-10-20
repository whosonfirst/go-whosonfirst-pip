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
go get -u "github.com/hashicorp/golang-lru"
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
p := pip.NewPointInPolygonSimple(source)

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

# Assuming you're filtering on placetype

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

There is also a standalone HTTP server for performing point-in-polygon lookups. It is instantiated with a `data` parameter and one or more "meta" CSV files, like this:

```
./bin/pip-server -data /usr/local/mapzen/whosonfirst-data/data/ -strict /usr/local/mapzen/whosonfirst-data/meta/wof-country-latest.csv /uslocal/mapzen/whosonfirst-data/meta/wof-neighbourhood-latest.csv 
indexed 50125 records in 64.023 seconds 
[placetype] country 219
[placetype] neighbourhood 49906
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
        "Id": 85633793,
        "Name": "United States",
        "Placetype": "country"
    },
    {
        "Id": 85865587,
        "Name": "Gowanus",
        "Placetype": "neighbourhood"
    }
]
```

There is an optional third `placetype` parameter which is a string (see also: [the list of valid Who's On First placetypes](https://github.com/whosonfirst/whosonfirst-placetypes)) that will limit the results to only records of a given placetype. Like this:

```
$> curl 'http://localhost:8080?latitude=40.677524&longitude=-73.987343&placetype=neighbourhood' | python -mjson.tool
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

You can enable strict placetype checking on the server-side by specifying the `-strict` flag. This will ensure that the placetype being specificed has actually been indexed, returning an error if not.

## Assumptions, caveats and known-knowns

### When we say `geojson` in the context of Go-typing

We are talking about the [go-whosonfirst-geojson](https://www.github.com/whosonfirst/go-whosonfirst-geojson) library.

### Speed and performance

This is how it works now:

1. We are using the [rtreego](https://www.github.com/dhconnelly/rtreego) library to do most of the heavy lifting and filtering.
2. Results from the rtreego `SearchIntersect` method are "inflated" and recast as geojson `WOFSpatial` object-interface-struct-things.
3. We are performing a final containment check on the results by reading each corresponding GeoJSON file and converting its geometry in to one or more [golang-geo](https://www.github.com/kellydunn/golang-geo) `Polygon` object-interface-struct-things. Each of these object-interface-struct-things calls its `Contains` method on an input coordinate.
4. If any given set of `Polygon` object-interface-struct-things contains more than 100 points it is cached using the [golang-lru](https://github.com/hashicorp/golang-lru) package.

### Caching

This is what it looks like reverse-geocoding a point on the island of Montéal against the set of all countries in Who's On First:

```
[debug] time to marshal /usr/local/mapzen/whosonfirst-data/data/856/330/41/85633041.geojson is 0.072996
[debug] time to convert geom to polygons (67631 points) is 0.007654
[cache] 85633041 because so many points (67631)
[debug] time to load polygons is 0.080718
[debug] time to check containment (true) after 10/382 possible iterations is 0.000007
[debug] time to marshal /usr/local/mapzen/whosonfirst-data/data/856/326/85/85632685.geojson is 0.957890
[debug] time to convert geom to polygons (469372 points) is 0.135788
[cache] 85632685 because so many points (469372)
[debug] time to load polygons is 1.093759
[debug] time to check containment (false) after 4800/4800 possible iterations is 0.005420
[debug] time to load polygons is 0.000003
[debug] time to check containment (false) after 75/75 possible iterations is 0.001028
[debug] contained: 1/3
[timings] 45.572744, -73.586295 (1 result)
[timing] intersects: 0.000030
[timing] inflate: 0.000001
[timing] contained: 1.181019

# this time loading polygons from cache

[debug] time to load polygons is 0.000003
[debug] time to check containment (true) after 10/382 possible iterations is 0.000006
[debug] time to load polygons is 0.000001
[debug] time to check containment (false) after 4800/4800 possible iterations is 0.005379
[debug] time to load polygons is 0.000001
[debug] time to check containment (false) after 75/75 possible iterations is 0.001023
[debug] contained: 1/3
[timings] 45.572744, -73.586295 (1 result)
[timing] intersects: 0.000025
[timing] inflate: 0.000001
[timing] contained: 0.006456
```

Some countries, like [New Zealand](https://whosonfirst.mapzen.com/spelunker/id/85633345/) are known to be problematic because they have an insanely large "ground truth" polygon, but the caching definitely helps. For example, reverse-geocoding `-40.357418,175.611481` looks like this:

```
[debug] time to marshal /usr/local/mapzen/whosonfirst-data/data/856/333/45/85633345.geojson is 5.419391
[debug] time to convert geom to polygons (3022193 points) is 0.326103
[cache] 85633345 because so many points (3022193)
[debug] time to load polygons is 5.745573
[debug] time to check containment (true) after 1524/5825 possible iterations is 0.020504
[debug] contained: 1/1
[timings] -40.357418, 175.611481 (1 result)
[timing] intersects: 0.000081
[timing] inflate: 0.000004
[timing] placetype: 0.000001
[timing] contained: 5.766121

# this time loading polygons from cache

[debug] time to load polygons is 0.000003
[debug] time to check containment (true) after 1524/5825 possible iterations is 0.020891
[debug] contained: 1/1
[timings] -40.357418, 175.611481 (1 result)
[timing] intersects: 0.000082
[timing] inflate: 0.000001
[timing] placetype: 0.000001
[timing] contained: 0.020952
```

So the amount of time it takes to perform the final point-in-polygon test is relatively constant but the difference between fetching the cached and uncached polygons to test is `0.000003` seconds versus `5.419391` so that's a thing.

There is a separate on-going process for [sorting out geometries in Who's On First](https://github.com/whosonfirst/whosonfirst-geometries) but on-going work is on-going. Whatever the case there is room for making this "Moar Faster".

### Memory usage

It is still possible, with enough concurrent requests all loading the countries files, to gobble up all the memory and fail unceremoniously. On the other hand, if you take countries and their monster geometries out of the mix everything seems fine:

```
$> siege -c 100 -i -f urls.txt
** SIEGE 3.0.5
** Preparing 100 concurrent users for battle.
The server is now under siege...^C
Lifting the server siege...      done.

Transactions:			17939 hits
Availability:			100.00 %
Elapsed time:			91.00 secs
Data transferred:		0.59 MB
Response time:			0.00 secs
Transaction rate:		197.13 trans/sec
Throughput:			0.01 MB/sec
Concurrency:			0.27
Successful transactions:	17939
Failed transactions:		0
Longest transaction:		0.11
Shortest transaction:		0.00
```

_The point of the above is that memory usage was low and constant._

But yeah. Something is going on with countries. And we should fix that...

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
* https://github.com/hashicorp/golang-lru
* https://www.github.com/whosonfirst/go-whosonfirst-geojson
* https://whosonfirst.mapzen.com/data/
