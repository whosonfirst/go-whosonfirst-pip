package main

import (
	"flag"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-pip"
	"os"
)

func main() {

	var source = flag.String("source", "", "The source directory where WOF data lives")

	flag.Parse()
	args := flag.Args()

	if *source == "" {
		panic("missing source")
	}

	_, err := os.Stat(*source)

	if os.IsNotExist(err) {
		panic("source does not exist")
	}

	p, p_err := pip.PointInPolygon(*source)

	if p_err != nil {
		panic(p_err)
	}

	for _, path := range args {
		// fmt.Println(path)
		p.IndexGeoJSONFile(path)
	}

	fmt.Printf("indexed %d records\n", p.Rtree.Size())

	lat := 37.791614
	lon := -122.392375

	fmt.Printf("get by lat lon %f, %f\n", lat, lon)

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

	simple, _ := p.GetByLatLon(lat, lon)

	for i, f := range simple {
		fmt.Printf("simple result #%d is %s\n", i, f.Name)
	}

}
