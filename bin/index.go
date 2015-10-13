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

	for i, r := range results {
		fmt.Printf("result #%d is %s\n", i, r)
	}		
}
