package main

import (
	"flag"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-pip"
)

func main() {

	flag.Parse()
	args := flag.Args()

	p := pip.PointInPolygon()

	for _, path := range args {
		// fmt.Println(path)
		p.IndexGeoJSONFile(path)
	}

	fmt.Printf("indexed %d records\n", p.Rtree.Size())

	lat := 37.791614
	lon := -122.392375

	fmt.Printf("get by lat lon %f, %f\n", lat, lon)

	results := p.GetByLatLon(lat, lon)
	inflated := p.InflateResults(results)

	for i, wof := range inflated {
		fmt.Printf("result #%d is %s\n", i, wof.Name)
	}

	fmt.Println("filter results by locality")

	filtered := p.FilterByPlacetype(inflated, "locality")

	for i, f := range filtered {
		fmt.Printf("filtered result #%d is %s\n", i, f.Name)
	}

}
