# go-whosonfirst-pip

An in-memory point-in-polygon (reverse geocoding) library for Who's On First data

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
go get -u "github.com/whosonfirst/go-whosonfirst-csv"
go get -u "github.com/whosonfirst/go-whosonfirst-logs"
go get -u "github.com/dhconnelly/rtreego"
go get -u "github.com/hashicorp/golang-lru"
go get -u "github.com/rcrowley/go-metrics"
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
results, _ := p.GetIntersectsByLatLon(lat, lon)

for i, r := range results {
	fmt.Printf("spatial result #%d is %v\n", i, r)
}

inflated, _ := p.InflateSpatialResults(results)

for i, wof := range inflated {
	fmt.Printf("wof result #%d is %s\n", i, wof.Name)
}

# Assuming you're filtering on placetype

filtered, _ := p.FilterByPlacetype(inflated, "locality")

for i, f := range filtered {
	fmt.Printf("filtered result #%d is %s\n", i, f.Name)
}

contained, _ := p.EnsureContained(lat, lon, inflated)

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

You can enable strict placetype checking on the server-side by specifying the `-strict` flag. This will ensure that the placetype being specificed has actually been indexed, returning an error if not. `pip-server` has many other option-knobs and they are:

```
$> ./bin/pip-server -help
Usage of ./bin/pip-server:
  -cache_size int
    	      The number of WOF records with large geometries to cache (default 1024)
  -cache_trigger int
    		 The minimum number of coordinates in a WOF record that will trigger caching (default 2000)
  -cors
	Enable CORS headers
  -data string
    	The data directory where WOF data lives, required
  -logs string
    	Where to write logs to disk
  -metrics string
    	   Where to write (@rcrowley go-metrics style) metrics to disk
  -metrics-as string
    	      Format metrics as... ? Valid options are "json" and "plain" (default "plain")
  -port int
    	The port number to listen for requests on (default 8080)
  -strict
	Enable strict placetype checking
  -verbose
	Enable verbose logging, or log level "info"
  -verboser
	Enable really verbose logging, or log level "debug"
```

## Metrics

This package uses Richard Crowley's [go-metrics](https://github.com/rcrowley/go-metrics) package to record general [memory statistics](https://golang.org/pkg/runtime/#MemStats) and a handful of [custom metrics](https://github.com/whosonfirst/go-whosonfirst-pip/blob/master/pip.go#L20-L32).

### Custom metrics

#### pip.reversegeo.lookups

The total number of reverse geocoding lookups. This is a `metrics.Counter`.

#### pip.geojson.unmarshaled

The total number of time any GeoJSON file has been unmarshaled. This is a `metrics.Counter` thingy.

#### pip.cache.hit

The number of times a record has been found in the LRU cache. This is a `metrics.Counter` thingy.

#### pip.cache.miss

The number of times a record has _not_ been found in the LRU cache. This is a `metrics.Counter` thingy.

#### pip.cache.set

The number of times a record has been added to the LRU cache. This is a `metrics.Counter` thingy.

#### pip.timer.reversegeo

The total amount of time to complete a reverse geocoding lookup. This is a `metrics.Timer` thingy.

#### pip.timer.unmarshal

The total amount of time to read and unmarshal a GeoJSON file from disk. This is a `metrics.Timer` thingy.

#### pip.timer.containment

The total amount of time to perform final raycasting intersection tests. This is a `metrics.Timer` thingy.

### Configuring metrics

If you are using the `pip` package in your own program you will need to tell the package where to send the metrics. You can do this by passing the following to the `SendMetricsTo` method:

* Anything that implements an `io.Writer` interface
* The frequency that metrics should be reported as represented by something that implements the `time.Duration` interface
* Either `plain` or `json` which map to the [metrics.Log](https://github.com/rcrowley/go-metrics/blob/master/log.go) and [metrics.JSON](https://github.com/rcrowley/go-metrics/blob/master/json.go) packages respectively

#### Example

```
m_file, m_err := os.OpenFile("metrics.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)

if m_err != nil {
	panic(m_err)
}

m_writer = io.MultiWriter(m_file)
_ = p.SendMetricsTo(m_writer, 60e9, "plain")
```

## Assumptions, caveats and known-knowns

### When we say `geojson` in the context of Go-typing

We are talking about the [go-whosonfirst-geojson](https://www.github.com/whosonfirst/go-whosonfirst-geojson) library.

### Speed and performance

This is how it works now:

1. We are using the [rtreego](https://www.github.com/dhconnelly/rtreego) library to do most of the heavy lifting and filtering.
2. Results from the rtreego `SearchIntersect` method are "inflated" and recast as geojson `WOFSpatial` object-interface-struct-things.
3. We are performing a final containment check on the results by reading each corresponding GeoJSON file using [go-whosonfirst-geojson](https://github.com/whosonfirst/go-whosonfirst-geojson) and calling the `Contains` method on each of the items returned by the `GeomToPolygon` method. What's _actually_ happening is that the GeoJSON geometry is being converted in to one or more [golang-geo](https://www.github.com/kellydunn/golang-geo) `Polygon` object-interface-struct-things. Each of these object-interface-struct-things calls its `Contains` method on an input coordinate.
4. If any given set of `Polygon` object-interface-struct-things contains more than `n` points (where `n` is defined by the `CacheTrigger` constructor thingy or the `cache_trigger` command line argument) it is cached using the [golang-lru](https://github.com/hashicorp/golang-lru) package.

### Caching

We are aggressively pre-caching large (or slow) GeoJSON files or GeoJSON files with large geometries in the LRU cache. As of this writing during the start-up process when we are building the Rtree any GeoJSON file that takes > 0.01 seconds to load is tested to see whether it has >= 2000 vertices. If it does then it is added to the LRU cache.

Both the size of the cache and the trigger (number of vertices) are required parameters when instatiating a `WOFPointInPolygon` object-interface-struct thing. Like this:

```
func NewPointInPolygon(source string, cache_size int, cache_trigger int, logger *log.WOFLogger) (*WOFPointInPolygon, error) {
     // ...
}
```

You should adjust these values to taste. If you are adding more records to the cache than you've allocated space for the package will emit warnings telling you that, during the start-up phase.

This is all to account for the fact that some countries, like [New Zealand](https://whosonfirst.mapzen.com/spelunker/id/85633345/) are known to be problematic because they have an insanely large "ground truth" polygon, but the caching definitely helps. For example, reverse-geocoding `-40.357418,175.611481` looks like this:

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
```

This is what things look like loading the same data from cache:

```
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

### Load testing

Individual reverse geocoding lookups are almost always sub-second responses. After unmarshaling GeoJSON files (which are cached) the bottleneck appears to be in the final raycasting intersection tests for anything that is a match in the Rtree and warnings are emitted for anything that takes longer than 0.5 seconds. Although there is room for improvement here (a more efficient raycasting, etc. ) this is mostly only a problem for countries and very large and fiddly cities as evidenced by our load-testing benchmarks.

```
$> siege -c 100 -i -f urls.txt
** SIEGE 3.0.5
** Preparing 100 concurrent users for battle.
The server is now under siege...^C
Lifting the server siege...      done.

Transactions:				57270 hits
Availability:				100.00 %
Elapsed time:				314.56 secs
Data transferred:			3.18 MB
Response time:				0.05 secs
Transaction rate:			182.06 trans/sec
Throughput:				0.01 MB/sec
Concurrency:				8.68
Successful transactions:		57270
Failed transactions:			0
Longest transaction:			1.70
Shortest transaction:			0.00

$> siege -c 500 -i -f urls.txt
** SIEGE 3.0.5
** Preparing 500 concurrent users for battle.
The server is now under siege...^C
Lifting the server siege...      done.

Transactions:				118034 hits
Availability:				99.98 %
Elapsed time:				475.11 secs
Data transferred:			6.56 MB
Response time:				1.47 secs
Transaction rate:			248.44 trans/sec
Throughput:				0.01 MB/sec
Concurrency:				365.65
Successful transactions:		118034
Failed transactions:			20
Longest transaction:			65.09
Shortest transaction:			0.03

$> siege -c 250 -i -f urls.txt
** SIEGE 3.0.5
** Preparing 250 concurrent users for battle.
The server is now under siege...^C
Lifting the server siege...      done.

Transactions:				96861 hits
Availability:				100.00 %
Elapsed time:				390.72 secs
Data transferred:			5.38 MB
Response time:				0.51 secs
Transaction rate:			247.90 trans/sec
Throughput:				0.01 MB/sec
Concurrency:				125.76
Successful transactions:		96861
Failed transactions:			0
Longest transaction:			4.07
Shortest transaction:			0.01

$> siege -c 300 -i -f urls-wk.txt 
siege aborted due to excessive socket failure; you
can change the failure threshold in $HOME/.siegerc

Transactions:				897266 hits
Availability:				99.85 %
Elapsed time:				3760.40 secs
Data transferred:			43.62 MB
Response time:				0.67 secs
Transaction rate:			238.61 trans/sec
Throughput:				0.01 MB/sec
Concurrency:				160.68
Successful transactions:		896961
Failed transactions:			1323
Longest transaction:			31.51
Shortest transaction:			0.01
```

### Memory usage

Memory usage will depend on the data that you've imported, obviously. In the past (before we cached things so aggressively) it was possible to send the `pip-server` in to an ungracious death spiral by hitting the server with too many concurrent requests that required it to load large country GeoJSON files.

Pre-caching files seems to account for this problem (see load testing stats above) but as with any service I'm sure there is still a way to overwhelm it. The good news is that in the testing we've done so far memory usage for the `pip-server` remains pretty constant regardless of the number of connections attempting to talk to it.

For a server loading all of the [countries](https://github.com/whosonfirst/whosonfirst-data/blob/master/meta/wof-country-latest.csv), [localities](https://github.com/whosonfirst/whosonfirst-data/blob/master/meta/wof-locality-latest.csv) and [neightbourhoods](https://github.com/whosonfirst/whosonfirst-data/blob/master/meta/wof-neighbourhood-latest.csv) in Who's On First these are the sort of numbers (measured in bytes) we're seeing as reported by the metrics package:

```
$> /bin/grep -A 1 runtime.MemStats.Alloc metrics.log
[pip-metrics] 23:39:13.978103 gauge runtime.MemStats.Alloc
[pip-metrics] 23:39:13.978107   value:       876122856

$> /bin/grep -A 1 runtime.MemStats.HeapInuse metrics.log
[pip-metrics] 23:39:13.977245 gauge runtime.MemStats.HeapInuse
[pip-metrics] 23:39:13.977249   value:       1273307136
```

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
* https://github.com/hashicorp/golang-lru
* https://github.com/rcrowley/go-metrics
* https://www.github.com/whosonfirst/go-whosonfirst-geojson
* https://whosonfirst.mapzen.com/data/
