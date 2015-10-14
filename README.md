# go-whosonfirst-pip

Expermimental point-in-polygon library for Who's On First documents

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

You can index individual GeoJSON files or Who's On First "meta" files, which are CSV files with pointers to individual Who's On First records.

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

`results` contains a list of `geojson.WOFSpatail` objects and `timings` contains a list of `pip.WOFPointInPolygonTiming` objects. 

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

### HTTP Ponies

There is also a standalone HTTP server for performing point-in-polygon lookups. It is instantiated with one or more "meta" CSV files, like this:

```
$> ./bin/pip-server -source /usr/local/mapzen/whosonfirst-data/data /usr/local/mapzen/whosonfirst-data/meta/wof-neighbourhood-latest.csv
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

### Speed and performance

This is how it works now:

1. We are using the [rtreego](https://www.github.com/dhconnelly/rtreego) library to do most of the heavy lifting and filtering
2. Results from the rtreego `SearchIntersect` method are "inflated" and recast as [wof-geojson](https://www.github.com/whosonfirst/go-whosonfirst-geojson) `WOFSpatial` objects
3. We are performing a final containment check on the results by reading the corresponding GeoJSON file and reading its geometry in to one or more [golang-geo](https://www.github.com/kellydunn/golang-geo) `Polygon` objects. Each of these objects calls its `Contains` method on an input coordinate.

This is how long it takes reverse-geocoding a point in Brooklyn, using an index of all the countries in Who's On First:

```
[timings] 40.677524,-73.987343 ()
[timing] intersects: 0.000030
[timing] inflate: 0.000000
[timing] contained: 0.115600
```

These numbers are still a bit vague and misleading. For example it's not clear (because it hasn't been measured yet) where most of the work in that 0.1 seconds is happening. Is it reading the GeoJSON file? It is converting the file's geometry in to Polygon objects? It is actually testing a single coordinate against a giant bag of coordinates? I don't know, yet.

## See also

* https://www.github.com/dhconnelly/rtreego
* https://www.github.com/kellydunn/golang-geo
* https://www.github.com/whosonfirst/go-whosonfirst-geojson