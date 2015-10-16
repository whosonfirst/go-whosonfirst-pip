package main

import (
	"flag"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-pip"
	"os"
	"time"
)

func main() {

	var source = flag.String("source", "", "The source directory where WOF data lives")
	var offset = flag.Int("offset", 0, "The (start by zero) offset at which the relative path for a record lives (THIS IS NOT A FEATURE)")

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

	t1 := time.Now()

	for _, path := range args {

		p.IndexMetaFile(path, *offset)
	}

	t2 := float64(time.Since(t1)) / 1e9

	fmt.Printf("indexed %d records in %.3f seconds \n", p.Rtree.Size(), t2)

	lat := 37.791614
	lon := -122.392375

	fmt.Printf("get by lat lon %f, %f\n", lat, lon)

	results, _ := p.GetByLatLon(lat, lon)

	for i, wof := range results {

		fmt.Printf("result #%d is %s\n", i, wof.Name)
	}
}
