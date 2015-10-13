package main

import (
       "encoding/csv"
	"flag"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-pip"
	"os"
	"io"
	"path"
)

func main() {

	var source = flag.String("csv", "", "The csv file to index")

	flag.Parse()

	if *source == "" {
		panic("missing source to sync")
	}

	_, err := os.Stat(*source)

	if os.IsNotExist(err) {
		panic("source does not exist")
	}

	p := pip.PointInPolygon()

	body, read_err := os.Open(*source)

	if read_err != nil {
	   panic(read_err)
	}

	r := csv.NewReader(body)

	for {
	      record, err := r.Read()

      	      if err == io.EOF {
	    	     break
	      }

    	      if err != nil {
	             panic(err)
	      }

	      rel_path := record[12]
	      abs_path := path.Join("/usr/local/mapzen/whosonfirst-data/data", rel_path)

	      _, err = os.Stat(abs_path)

	      if os.IsNotExist(err) {
		continue
	      }

	      fmt.Println(abs_path)

	      p.IndexGeoJSONFile(abs_path)
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
}
