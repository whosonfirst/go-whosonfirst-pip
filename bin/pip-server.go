package main

import (
	"encoding/json"
	"flag"
	"fmt"
	log "github.com/whosonfirst/go-whosonfirst-log"
	pip "github.com/whosonfirst/go-whosonfirst-pip"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {

	var port = flag.Int("port", 8080, "The port number to listen for requests on")
	var data = flag.String("data", "", "The data directory where WOF data lives, required")
	var cache_size = flag.Int("cache_size", 1024, "The number of WOF records with large geometries to cache")
	var cache_trigger = flag.Int("cache_trigger", 2000, "The minimum number of coordinates in a WOF record that will trigger caching")
	var strict = flag.Bool("strict", false, "Enable strict placetype checking")
	var logs = flag.String("logs", "", "Where to write logs to disk")
	var metrics = flag.String("metrics", "", "Where to write (@rcrowley go-metrics style) metrics to disk")
	var format = flag.String("metrics-as", "plain", "Format metrics as... ? Valid options are \"json\" and \"plain\"")
	var verbose = flag.Bool("verbose", false, "Enable verbose logging, or log level \"info\"")
	var verboser = flag.Bool("verboser", false, "Enable really verbose logging, or log level \"debug\"")

	flag.Parse()
	args := flag.Args()

	if *data == "" {
		panic("missing data")
	}

	_, err := os.Stat(*data)

	if os.IsNotExist(err) {
		panic("data does not exist")
	}

	loglevel := "status"

	if *verbose {
		loglevel = "info"
	}

	if *verboser {
		loglevel = "debug"
	}

	var l_writer io.Writer
	var m_writer io.Writer

	l_writer = io.MultiWriter(os.Stdout)

	if *logs != "" {

		l_file, l_err := os.OpenFile(*logs, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)

		if l_err != nil {
			panic(l_err)
		}

		l_writer = io.MultiWriter(os.Stdout, l_file)
	}

	logger := log.NewWOFLogger(l_writer, "[pip-server] ", loglevel)

	p, p_err := pip.NewPointInPolygon(*data, *cache_size, *cache_trigger, logger)

	if p_err != nil {
		panic(p_err)
	}

	if *metrics != "" {

		m_file, m_err := os.OpenFile(*metrics, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)

		if m_err != nil {
			panic(m_err)
		}

		m_writer = io.MultiWriter(m_file)
		_ = p.SendMetricsTo(m_writer, 60e9, *format)
	}

	t1 := time.Now()

	for _, path := range args {
		p.IndexMetaFile(path)
	}

	t2 := float64(time.Since(t1)) / 1e9

	p.Logger.Status("indexed %d records in %.3f seconds", p.Rtree.Size(), t2)

	for pt, count := range p.Placetypes {
		p.Logger.Status("indexed %s: %d", pt, count)
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

		if placetype != "" {

			if *strict && !p.IsKnownPlacetype(placetype) {
				http.Error(rsp, "Unknown placetype", http.StatusBadRequest)
				return
			}
		}

		results, timings := p.GetByLatLonForPlacetype(lat, lon, placetype)

		count := len(results)
		ttp := 0.0

		for _, t := range timings {
			ttp += t.Duration
		}

		if placetype != "" {
			p.Logger.Debug("time to reverse geocode %f, %f @%s: %d results in %f seconds ", lat, lon, placetype, count, ttp)
		} else {
			p.Logger.Debug("time to reverse geocode %f, %f: %d results in %f seconds ", lat, lon, count, ttp)
		}

		js, err := json.Marshal(results)

		if err != nil {
			http.Error(rsp, err.Error(), http.StatusInternalServerError)
			return
		}

		rsp.Header().Set("Content-Type", "application/json")
		rsp.Write(js)
	}

	str_port := fmt.Sprintf(":%d", *port)

	http.HandleFunc("/", handler)
	http.ListenAndServe(str_port, nil)
}
