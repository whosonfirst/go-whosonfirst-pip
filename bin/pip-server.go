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

	t1 := time.Now()

	for _, path := range args {
		p.IndexMetaFile(path, 12)
	}

	t2 := float64(time.Since(t1)) / 1e9

	fmt.Printf("indexed %d records in %.3f seconds \n", p.Rtree.Size(), t2)

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

		// please validate placetype here...

		if placetype == "" {
			results, timings = p.GetByLatLon(lat, lon)
		} else {
			results, timings = p.GetByLatLonForPlacetype(lat, lon, placetype)
		}

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
