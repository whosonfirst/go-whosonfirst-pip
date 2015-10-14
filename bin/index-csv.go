package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-pip"
	"io"
	"os"
	"path"
	"time"
)

// sudo do not panic but return an actual error thingy
// (20151013/thisisaaronland)

// Please add me to pip.go as a IndexMetaFile method
// (20151013/thisisaaronland)

func IndexCSVFile(idx *pip.WOFPointInPolygon, csv_file string, source string, offset int) error {

	body, read_err := os.Open(csv_file)

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

		// sudo my kingdom for a DictReader in Go...
		// (20151013/thisisaaronland)

		rel_path := record[offset]
		abs_path := path.Join(source, rel_path)

		_, err = os.Stat(abs_path)

		if os.IsNotExist(err) {
			continue
		}

		idx.IndexGeoJSONFile(abs_path)
	}

	return nil
}

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

	p := pip.PointInPolygon(*source)

	t1 := time.Now()

	for _, path := range args {

		IndexCSVFile(p, path, *source, *offset)
	}

	t2 := float64(time.Since(t1)) / 1e9

	fmt.Printf("indexed %d records in %.3f seconds \n", p.Rtree.Size(), t2)

	lat := 37.791614
	lon := -122.392375

	fmt.Printf("get by lat lon %f, %f\n", lat, lon)

	results := p.GetByLatLon(lat, lon)
	inflated := p.InflateResults(results)

	for i, wof := range inflated {

		fmt.Printf("result #%d is %s\n", i, wof.Name)
	}
}
