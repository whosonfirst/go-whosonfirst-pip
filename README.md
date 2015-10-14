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
p.IndexGeoJSONFile(path)
```

### Simple

```
simple, timings := p.GetByLatLon(lat, lon)

for i, f := range simple {
	fmt.Printf("simple result #%d is %s\n", i, f.Name)
}

for _, t := range timings {
        fmt.Printf("[timing] %s: %f\n", t.Event, t.Duration)
}
```

### What's going on under the hood

```
results := p.GetIntersectsByLatLon(lat, lon)
inflated := p.InflateSpatialResults(results)

for i, wof := range inflated {
	fmt.Printf("result #%d is %s\n", i, wof.Name)
}

fmt.Println("filter results by locality")

filtered := p.FilterByPlacetype(inflated, "locality")

for i, f := range filtered {
	fmt.Printf("filtered result #%d is %s\n", i, f.Name)
}

fmt.Println("ensure contained")

contained := p.EnsureContained(lat, lon, inflated)

for i, f := range contained {
	fmt.Printf("contained result #%d is %s\n", i, f.Name)
}

```

### HTTP Ponies

```
$> ./bin/pip-server -source /usr/local/mapzen/whosonfirst-data/data /usr/local/mapzen/whosonfirst-data/meta/wof-neighbourhood-latest.csv
```

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

```
[timings] 40.677524,-73.987343 ()
[timing] intersects: 0.000030
[timing] inflate: 0.000000
[timing] contained: 0.115600
```

## See also

* https://www.github.com/dhconnelly/rtreego
* https://www.github.com/kellydunn/golang-geo
* https://www.github.com/whosonfirst/go-whosonfirst-geojson