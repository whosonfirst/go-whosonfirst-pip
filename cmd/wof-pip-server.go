package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/facebookgo/grace/gracehttp"
	log "github.com/whosonfirst/go-whosonfirst-log"
	pip "github.com/whosonfirst/go-whosonfirst-pip"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"
)

func main() {

	var host = flag.String("host", "localhost", "The hostname to listen for requests on")
	var port = flag.Int("port", 8080, "The port number to listen for requests on")
	var data = flag.String("data", "", "The data directory where WOF data lives, required")
	var cache_all = flag.Bool("cache_all", false, "Just cache everything, regardless of size")
	var cache_size = flag.Int("cache_size", 1024, "The number of WOF records with large geometries to cache")
	var cache_trigger = flag.Int("cache_trigger", 2000, "The minimum number of coordinates in a WOF record that will trigger caching")
	var strict = flag.Bool("strict", false, "Enable strict placetype checking")
	var loglevel = flag.String("loglevel", "info", "Log level for reporting")
	var logs = flag.String("logs", "", "Where to write logs to disk")
	var metrics = flag.String("metrics", "", "Where to write (@rcrowley go-metrics style) metrics to disk")
	var format = flag.String("metrics-as", "plain", "Format metrics as... ? Valid options are \"json\" and \"plain\"")
	var cors = flag.Bool("cors", false, "Enable CORS headers")
	var procs = flag.Int("procs", (runtime.NumCPU() * 2), "The number of concurrent processes to clone data with")
	var pidfile = flag.String("pidfile", "", "Where to write a PID file for wof-pip-server. If empty the PID file will be written to wof-pip-server.pid in the current directory")

	flag.Parse()
	args := flag.Args()

	if *data == "" {
		panic("missing data")
	}

	_, err := os.Stat(*data)

	if os.IsNotExist(err) {
		panic("data does not exist")
	}

	runtime.GOMAXPROCS(*procs)

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

	logger := log.NewWOFLogger("[wof-pip-server] ")
	logger.AddLogger(l_writer, *loglevel)

	if *cache_all {

		*cache_size = 0
		*cache_trigger = 1

		mu := new(sync.Mutex)
		wg := new(sync.WaitGroup)

		for _, path := range args {

			wg.Add(1)

			go func(path string) {
				defer wg.Done()

				count := 0

				fh, err := os.Open(path)

				if err != nil {
					logger.Error("failed to open %s for reading, because %v", path, err)
					os.Exit(1)
				}

				scanner := bufio.NewScanner(fh)

				for scanner.Scan() {
					count += 1
				}

				mu.Lock()
				*cache_size += count
				mu.Unlock()

			}(path)
		}

		wg.Wait()

		logger.Status("set cache_size to %d and cache_trigger to %d", *cache_size, *cache_trigger)
	}

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

	indexing := true
	ch := make(chan bool)

	go func() {
		<-ch
		indexing = false
	}()

	go func() {

		if *pidfile == "" {

			cwd, err := os.Getwd()

			if err != nil {
				panic(err)
			}

			fname := fmt.Sprintf("%s.pid", os.Args[0])

			*pidfile = filepath.Join(cwd, fname)
		}

		fh, err := os.Create(*pidfile)

		if err != nil {
			panic(err)
		}

		defer fh.Close()

		t1 := time.Now()

		for _, path := range args {
			p.IndexMetaFile(path)
		}

		t2 := float64(time.Since(t1)) / 1e9
		p.Logger.Status("indexed %d records in %.3f seconds", p.Rtree.Size(), t2)

		/*
			for pt, count := range p.Placetypes {
				p.Logger.Status("indexed %s: %d", pt, count)
			}
		*/

		pid := os.Getpid()
		strpid := strconv.Itoa(pid)

		fh.Write([]byte(strpid))

		ch <- true
	}()

	handler := func(rsp http.ResponseWriter, req *http.Request) {

		if indexing == true {
			http.Error(rsp, "indexing records", http.StatusServiceUnavailable)
			return
		}

		query := req.URL.Query()

		str_lat := query.Get("latitude")
		str_lon := query.Get("longitude")
		placetype := query.Get("placetype")
		excluded := query["exclude"] // see the way we're accessing the map directly to get a list? yeah, that

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

		filters := pip.WOFPointInPolygonFilters{}

		if placetype != "" {

			if *strict && !p.IsKnownPlacetype(placetype) {
				http.Error(rsp, "Unknown placetype", http.StatusBadRequest)
				return
			}

			filters["placetype"] = placetype
		}

		for _, what := range excluded {

			if what == "deprecated" || what == "superseded" {
				filters[what] = false
			}
		}

		results, timings := p.GetByLatLonFiltered(lat, lon, filters)

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

		// maybe this although it seems like it adds functionality for a lot of
		// features this server does not need - https://github.com/rs/cors
		// (20151022/thisisaaronland)

		if *cors {
			rsp.Header().Set("Access-Control-Allow-Origin", "*")
		}

		rsp.Header().Set("Content-Type", "application/json")
		rsp.Write(js)
	}

	endpoint := fmt.Sprintf("%s:%d", *host, *port)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)

	gracehttp.Serve(&http.Server{Addr: endpoint, Handler: mux})

	os.Exit(0)
}
