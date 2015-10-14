package main

import (
	"flag"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-pip"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
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

	p := pip.PointInPolygon(*source)

	for _, path := range args {
		// this is index-csv and needs to be made
		// a package method (20151014/thisisaaronland)
		// p.IndexMetaFile(path, offset)
		p.IndexGeoJSONFile(path)
	}

	fmt.Printf("indexed %d records\n", p.Rtree.Size())

	handler := func(rsp http.ResponseWriter, req *http.Request) {

		query := req.URL.Query()
		// fmt.Printf("%v\n", query)

		str_lat := query.Get("latitude")
		str_lon := query.Get("longitude")

		lat, _ := strconv.ParseFloat(str_lat, 64)
		lon, _ := strconv.ParseFloat(str_lon, 64)

		results := p.GetByLatLon(lat, lon)

		js, err := json.Marshal(results)

		if err != nil {
		   http.Error(rsp, err.Error(), http.StatusInternalServerError)
		   return
		}

		rsp.Header().Set("Content-Type", "application/json")
		rsp.Write(js)
	}

	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
