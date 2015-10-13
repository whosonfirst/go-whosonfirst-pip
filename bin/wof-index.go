package main

import (
	"flag"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-crawl"
	"github.com/whosonfirst/go-whosonfirst-pip"
	"os"
)

func main() {

	var root = flag.String("root", "", "The directory to sync")

	flag.Parse()

	if *root == "" {
		panic("missing root to sync")
	}

	_, err := os.Stat(*root)

	if os.IsNotExist(err) {
		panic("root does not exist")
	}

	p := pip.PointInPolygon()

	c := crawl.NewCrawler(*root)

	callback := func(path string, info os.FileInfo) error {

		if info.IsDir() {
			return nil
		}

		fmt.Printf("index %s\n", path)
		p.IndexGeoJSONFile(path)

		return nil
	}

	c.Crawl(callback)
		
	fmt.Printf("indexed %d records\n", p.Rtree.Size())

	lat := 37.791614
	lon := -122.392375

	fmt.Printf("get by lat lon %f, %f\n", lat, lon)

	results := p.GetByLatLon(lat, lon)
	inflated := p.InflateResults(results)

	for i, wof := range inflated {

		fmt.Printf("result #%d is %s\n", i, wof.Name)
	}		
}
