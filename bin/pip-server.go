package main

import (
	"encoding/json"
	_ "expvar"
	"flag"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-geojson"
	"github.com/whosonfirst/go-whosonfirst-pip"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {

	var data = flag.String("data", "", "The data directory where WOF data lives")
	var strict = flag.Bool("strict", false, "Enable strict placetype checking")

	flag.Parse()
	args := flag.Args()

	if *data == "" {
		panic("missing data")
	}

	_, err := os.Stat(*data)

	if os.IsNotExist(err) {
		panic("data does not exist")
	}

	p, p_err := pip.PointInPolygon(*data)

	if p_err != nil {
		panic(p_err)
	}

	t1 := time.Now()

	for _, path := range args {
		p.IndexMetaFile(path, 12)
	}

	t2 := float64(time.Since(t1)) / 1e9

	fmt.Printf("indexed %d records in %.3f seconds \n", p.Rtree.Size(), t2)

	for pt, count := range p.Placetypes {
	    fmt.Printf("[placetype] %s %d\n", pt, count)
	}

	handler := func(rsp http.ResponseWriter, req *http.Request) {

		query := req.URL.Query()

		str_lat := query.Get("latitude")
		str_lon := query.Get("longitude")
		placetype := query.Get("placetype")

		if str_lat == "" {
			http.Error(rsp, "Missing latitude parameter", http.StatusBadRequest)
			return
		}

		if str_lon == "" {
			http.Error(rsp, "Missing longitude parameter", http.StatusBadRequest)
			return
		}

		lat, lat_err := strconv.ParseFloat(str_lat, 64)
		lon, lon_err := strconv.ParseFloat(str_lon, 64)

		if lat_err != nil {
			http.Error(rsp, "Invalid latitude parameter", http.StatusBadRequest)
			return
		}

		if lon_err != nil {
			http.Error(rsp, "Invalid longitude parameter", http.StatusBadRequest)
			return
		}

		if lat > 90.0 || lat < -90.0 {
			http.Error(rsp, "E_IMPOSSIBLE_LATITUDE", http.StatusBadRequest)
			return
		}

		if lon > 180.0 || lon < -180.0 {
			http.Error(rsp, "E_IMPOSSIBLE_LONGITUDE", http.StatusBadRequest)
			return
		}

		results := make([]*geojson.WOFSpatial, 0)
		timings := make([]*pip.WOFPointInPolygonTiming, 0)

		if placetype != "" {

			if *strict && !p.IsKnownPlacetype(placetype) {
				http.Error(rsp, "Unknown placetype", http.StatusBadRequest)
				return
			}
		}

		results, timings = p.GetByLatLonForPlacetype(lat, lon, placetype)

		count := len(results)

		fmt.Printf("[timings] %f, %f (%d results)\n", lat, lon, count)

		for _, t := range timings {
			fmt.Printf("[timing] %s: %f\n", t.Event, t.Duration)
		}

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
