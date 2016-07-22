package pip

import (
	_ "fmt"
	rtreego "github.com/dhconnelly/rtreego"
	lru "github.com/hashicorp/golang-lru"
	metrics "github.com/rcrowley/go-metrics"
	csv "github.com/whosonfirst/go-whosonfirst-csv"
	geojson "github.com/whosonfirst/go-whosonfirst-geojson"
	log "github.com/whosonfirst/go-whosonfirst-log"
	utils "github.com/whosonfirst/go-whosonfirst-utils"
	"io"
	golog "log"
	"os"
	"path"
	"sync"
	"time"
)

type WOFPointInPolygonMetrics struct {
	Registry        *metrics.Registry
	CountUnmarshal  *metrics.Counter
	CountCacheHit   *metrics.Counter
	CountCacheMiss  *metrics.Counter
	CountCacheSet   *metrics.Counter
	CountLookups    *metrics.Counter
	TimeToUnmarshal *metrics.Timer
	TimeToIntersect *metrics.Timer
	TimeToInflate   *metrics.Timer
	TimeToContain   *metrics.Timer
	TimeToProcess   *metrics.Timer
}

type WOFPointInPolygonFilters map[string]interface{} // these get expanded in func (p WOFPointInPolygon) Filter

func NewPointInPolygonMetrics() *WOFPointInPolygonMetrics {

	registry := metrics.NewRegistry()

	cnt_lookups := metrics.NewCounter()
	cnt_unmarshal := metrics.NewCounter()
	cnt_cache_hit := metrics.NewCounter()
	cnt_cache_miss := metrics.NewCounter()
	cnt_cache_set := metrics.NewCounter()

	tm_unmarshal := metrics.NewTimer()
	tm_intersect := metrics.NewTimer()
	tm_inflate := metrics.NewTimer()
	tm_contain := metrics.NewTimer()
	tm_process := metrics.NewTimer()

	registry.Register("pip.reversegeo.lookups", cnt_lookups)
	registry.Register("pip.geojson.unmarshaled", cnt_unmarshal)
	registry.Register("pip.cache.hit", cnt_cache_hit)
	registry.Register("pip.cache.miss", cnt_cache_miss)
	registry.Register("pip.cache.set", cnt_cache_set)
	registry.Register("pip.timer.reversegeo", tm_process)
	registry.Register("pip.timer.unmarshal", tm_unmarshal)
	// registry.Register("time-to-intersect", tm_intersect)
	// registry.Register("time-to-inflate", tm_inflate)
	registry.Register("pip.timer.containment", tm_contain)

	m := WOFPointInPolygonMetrics{
		Registry:        &registry,
		CountLookups:    &cnt_lookups,
		CountUnmarshal:  &cnt_unmarshal,
		CountCacheHit:   &cnt_cache_hit,
		CountCacheMiss:  &cnt_cache_miss,
		CountCacheSet:   &cnt_cache_set,
		TimeToUnmarshal: &tm_unmarshal,
		TimeToIntersect: &tm_intersect,
		TimeToInflate:   &tm_inflate,
		TimeToContain:   &tm_contain,
		TimeToProcess:   &tm_process,
	}

	metrics.RegisterRuntimeMemStats(registry)
	go metrics.CaptureRuntimeMemStats(registry, 10e9)

	return &m
}

type WOFPointInPolygonTiming struct {
	Event    string
	Duration float64
}

func NewWOFPointInPolygonTiming(event string, d time.Duration) *WOFPointInPolygonTiming {

	df := float64(d) / 1e9

	t := WOFPointInPolygonTiming{Event: event, Duration: df}
	return &t

}

type WOFPointInPolygon struct {
	Rtree        *rtreego.Rtree
	Cache        *lru.Cache
	CacheSize    int
	CacheTrigger int
	Source       string
	Placetypes   map[string]int
	Metrics      *WOFPointInPolygonMetrics
	Logger       *log.WOFLogger
}

func NewPointInPolygonSimple(source string) (*WOFPointInPolygon, error) {

	cache_size := 100
	cache_trigger := 1000

	logger := log.NewWOFLogger("[wof-pip-server]")
	logger.AddLogger(os.Stdout, "debug")

	return NewPointInPolygon(source, cache_size, cache_trigger, logger)
}

func NewPointInPolygon(source string, cache_size int, cache_trigger int, logger *log.WOFLogger) (*WOFPointInPolygon, error) {

	rtree := rtreego.NewTree(2, 25, 50)

	cache, err := lru.New(cache_size)

	if err != nil {
		return nil, err
	}

	metrics := NewPointInPolygonMetrics()

	placetypes := make(map[string]int)

	pip := WOFPointInPolygon{
		Rtree:        rtree,
		Source:       source,
		Cache:        cache,
		CacheSize:    cache_size,
		CacheTrigger: cache_trigger,
		Placetypes:   placetypes,
		Metrics:      metrics,
		Logger:       logger,
	}

	return &pip, nil
}

func (p WOFPointInPolygon) SendMetricsTo(w io.Writer, d time.Duration, format string) bool {

	var r metrics.Registry
	r = *p.Metrics.Registry

	if format == "plain" {
		l := golog.New(w, "[pip-metrics] ", golog.Lmicroseconds)
		go metrics.Log(r, d, l)
		return true
	} else if format == "json" {
		go metrics.WriteJSON(r, d, w)
		return true
	} else {
		p.Logger.Warning("unable to send metrics anywhere, because what is '%s'", format)
		return false
	}
}

func (p WOFPointInPolygon) IndexGeoJSONFile(path string) error {

	p.Logger.Debug("index %s", path)

	t := time.Now()

	feature, parse_err := p.LoadGeoJSON(path)

	d := time.Since(t)

	if parse_err != nil {
		return parse_err
	}

	index_err := p.IndexGeoJSONFeature(feature)

	if index_err != nil {
		return index_err
	}

	ttl := float64(d) / 1e9

	if ttl > 0.01 {
		p.Logger.Warning("scheduling %s for pre-caching because its time to load exceeds 0.01 seconds: %f", path, ttl)
		go p.LoadPolygonsForFeature(feature)
	}

	return nil
}

func (p WOFPointInPolygon) IndexGeoJSONFeature(feature *geojson.WOFFeature) error {

	body := feature.Body()

	geom_type, ok := body.Path("geometry.type").Data().(string)

	if ok && geom_type == "Point" {
		p.Logger.Warning("feature is a Point type so I am ignoring it...")
		return nil
	}

	spatial, spatial_err := feature.EnSpatialize()

	if spatial_err != nil {

		p.Logger.Error("failed to enspatialize feature, because %s", spatial_err)
		return spatial_err
	}

	return p.IndexSpatialFeature(spatial)
}

func (p WOFPointInPolygon) IndexSpatialFeature(spatial *geojson.WOFSpatial) error {

	pt := spatial.Placetype

	_, ok := p.Placetypes[pt]

	if ok {
		p.Placetypes[pt] += 1
	} else {
		p.Placetypes[pt] = 1
	}

	p.Rtree.Insert(spatial)

	return nil
}

func (p WOFPointInPolygon) IndexMetaFile(csv_file string) error {

	reader, reader_err := csv.NewDictReaderFromPath(csv_file)

	if reader_err != nil {
		p.Logger.Error("failed to create CSV reader , because %s", reader_err)
		return reader_err
	}

	// It is tempting to think that we could fan this out and process each row/file
	// concurrently but that will make the Rtree sad... (20151020/thisisaaronland)

	for {
		row, err := reader.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			p.Logger.Error("failed to parse CSV row , because %s", err)
			return err
		}

		rel_path, ok := row["path"]

		if ok != true {
			p.Logger.Warning("CSV row is missing a 'path' column")
			continue
		}

		abs_path := path.Join(p.Source, rel_path)

		_, err = os.Stat(abs_path)

		if os.IsNotExist(err) {
			p.Logger.Error("'%s' does not exist", abs_path)
			continue
		}

		index_err := p.IndexGeoJSONFile(abs_path)

		if index_err != nil {
			p.Logger.Error("failed to index '%s', because %s", abs_path, index_err)
			return index_err
		}
	}

	return nil
}

func (p WOFPointInPolygon) GetIntersectsByLatLon(lat float64, lon float64) ([]rtreego.Spatial, time.Duration) {

	// Error checking on rect?

	pt := rtreego.Point{lon, lat}
	rect, _ := rtreego.NewRect(pt, []float64{0.0001, 0.0001}) // how small can I make this?

	return p.GetIntersectsByRect(rect)
}

func (p WOFPointInPolygon) GetIntersectsByBoundingBox(swlat float64, swlon float64, nelat float64, nelon float64) ([]rtreego.Spatial, time.Duration) {

	// Error checking on rect?

	llat := nelat - swlat
	llon := nelon - swlon

	pt := rtreego.Point{swlon, swlat}
	rect, _ := rtreego.NewRect(pt, []float64{llon, llat})

	return p.GetIntersectsByRect(rect)
}

func (p WOFPointInPolygon) GetIntersectsByRect(rect *rtreego.Rect) ([]rtreego.Spatial, time.Duration) {

	t := time.Now()

	results := p.Rtree.SearchIntersect(rect)

	d := time.Since(t)

	var tm metrics.Timer
	tm = *p.Metrics.TimeToIntersect
	go tm.Update(d)

	return results, d
}

// maybe just merge this above - still unsure (20151013/thisisaaronland)

func (p WOFPointInPolygon) InflateSpatialResults(results []rtreego.Spatial) ([]*geojson.WOFSpatial, time.Duration) {

	t := time.Now()

	inflated := make([]*geojson.WOFSpatial, 0)

	for _, r := range results {

		// https://golang.org/doc/effective_go.html#interface_conversions

		wof := r.(*geojson.WOFSpatial)
		inflated = append(inflated, wof)
	}

	d := time.Since(t)

	var tm metrics.Timer
	tm = *p.Metrics.TimeToInflate
	go tm.Update(d)

	return inflated, d
}

func (p WOFPointInPolygon) GetByLatLon(lat float64, lon float64) ([]*geojson.WOFSpatial, []*WOFPointInPolygonTiming) {

	filters := WOFPointInPolygonFilters{}
	return p.GetByLatLonFiltered(lat, lon, filters)
}

// deprecated – just use GetByLatLonFiltered (20160722/thisisaaronland)

func (p WOFPointInPolygon) GetByLatLonForPlacetype(lat float64, lon float64, placetype string) ([]*geojson.WOFSpatial, []*WOFPointInPolygonTiming) {

	p.Logger.Warning("WOFPointInPolygon.GetByLatLonForPlacetype is deprecated, please user WOFPointInPolygon.GetByLatLonFiltered instead")

	filters := WOFPointInPolygonFilters{}
	filters["placetype"] = placetype

	return p.GetByLatLonFiltered(lat, lon, filters)
}

func (p WOFPointInPolygon) GetByLatLonFiltered(lat float64, lon float64, filters WOFPointInPolygonFilters) ([]*geojson.WOFSpatial, []*WOFPointInPolygonTiming) {

	var c metrics.Counter
	c = *p.Metrics.CountLookups
	go c.Inc(1)

	t := time.Now()

	timings := make([]*WOFPointInPolygonTiming, 0)

	intersects, duration := p.GetIntersectsByLatLon(lat, lon)
	timings = append(timings, NewWOFPointInPolygonTiming("intersects", duration))

	inflated, duration := p.InflateSpatialResults(intersects)
	timings = append(timings, NewWOFPointInPolygonTiming("inflate", duration))

	// See what's going on here? We are filtering by placetype before
	// do a final point-in-poly lookup so we don't try to load country
	// records while only searching for localities

	filtered, duration := p.Filter(inflated, filters)
	timings = append(timings, NewWOFPointInPolygonTiming("filter", duration))

	contained, duration := p.EnsureContained(lat, lon, filtered)
	timings = append(timings, NewWOFPointInPolygonTiming("contain", duration))

	d := time.Since(t)

	var tm metrics.Timer
	tm = *p.Metrics.TimeToProcess
	go tm.Update(d)

	ttp := float64(d) / 1e9

	if ttp > 0.5 {
		p.Logger.Warning("time to process %f,%f (%v) exceeds 0.5 seconds: %f", lat, lon, filters, ttp)

		for _, t := range timings {
			p.Logger.Info("[%s] %f", t.Event, t.Duration)
		}
	}

	return contained, timings
}

// deprecated - just use Filter (20160722/thisisaaronland)

func (p WOFPointInPolygon) FilterByPlacetype(results []*geojson.WOFSpatial, placetype string) ([]*geojson.WOFSpatial, time.Duration) {

	p.Logger.Warning("WOFPointInPolygon.FilterByPlacetype is deprecated, please user WOFPointInPolygon.Filter instead")

	filters := WOFPointInPolygonFilters{}
	filters["placetype"] = placetype

	return p.Filter(results, filters)
}

func (p WOFPointInPolygon) Filter(results []*geojson.WOFSpatial, filters WOFPointInPolygonFilters) ([]*geojson.WOFSpatial, time.Duration) {

	t := time.Now()

	filtered := make([]*geojson.WOFSpatial, 0)

	for _, r := range results {

		pt, ok := filters["placetype"]

		if ok && pt.(string) != r.Placetype {
			p.Logger.Debug("placetype filter failed, expected '%s' but got '%s'", pt, r.Placetype)
			continue
		}

		deprecated, ok := filters["deprecated"]

		if ok && deprecated.(bool) != r.Deprecated {
			p.Logger.Debug("deprecated filter failed, expected %t but got %t", deprecated, r.Deprecated)
			continue
		}

		superseded, ok := filters["superseded"]

		if ok && superseded.(bool) != r.Superseded {
			p.Logger.Debug("superseded filter failed, expected %t but got %t", superseded, r.Superseded)
			continue
		}

		filtered = append(filtered, r)
	}

	d := time.Since(t)

	return filtered, d
}

func (p WOFPointInPolygon) EnsureContained(lat float64, lon float64, results []*geojson.WOFSpatial) ([]*geojson.WOFSpatial, time.Duration) {

	// Okay - this isn't super complicated but it might look a bit scary
	// We're using a WaitGroup to process each possible result *and* we
	// are using (n) sub WaitGroups to process every polygon for each result
	// separately (20151020/thisisaaronland)

	// See also: https://talks.golang.org/2012/concurrency.slide#46
	// This is not what we're doing but it's essentially what the WaitGroups
	// implement but with a different syntax/pattern

	wg := new(sync.WaitGroup)

	/*
		Matt Amos [11:57]
		wow. i just found something unexpected: when i move `wg.Add(1)` out of the loop and change it to `wg.Add(1000000)`
		the runtime goes up from 600ms to 3s!

		Aaron Cope [11:58]
		that’s… a thing

		[11:58]
		it might also explain some lag that I’ve never been able to account for

		[11:58]
		I guess maybe Go is starting 1M “things” in the background, maybe?
	*/

	mu := new(sync.Mutex)

	contained := make([]*geojson.WOFSpatial, 0)
	// timings := make([]*WOFPointInPolygonTiming, 0)

	t := time.Now()

	for _, wof := range results {

		wg.Add(1)

		wg_ensure := func(wof *geojson.WOFSpatial) {

			defer wg.Done()

			polygons, err := p.LoadPolygons(wof)

			if err != nil {
				p.Logger.Error("failed to load polygons for %d, because %v", wof.Id, err)
				return
			}

			is_contained := false

			wg2 := new(sync.WaitGroup)

			for _, poly := range polygons {

				wg2.Add(1)

				wg_contains := func(p *geojson.WOFPolygon, lt float64, ln float64) {

					defer wg2.Done()

					if p.Contains(lt, ln) {
						is_contained = true
					}
				}

				go wg_contains(poly, lat, lon)
			}

			wg2.Wait()

			// All done checking the polygons - are we contained?

			// d2 := time.Since(t2)
			// contain_event := fmt.Sprintf("contain %d (%d/%d iterations, %d points)", id, iters, count, points)
			// timings = append(timings, NewWOFPointInPolygonTiming(contain_event, d2))

			/*

				See this? This is important. Specifically the part where we are locking
				updates to the 'contained' array. Which makes total sense obviously except
				for the part where I completely spaced on this (despite all the chatter and
				error checking around the waitgroups aboves...) and hilarity inevitably
				ensued. So you know, don't do what I did... (20160112/thisisaaronland)

				https://github.com/whosonfirst/go-whosonfirst-pip/commit/986e527dbe9e62915757489db7c70d5140c53629
				https://github.com/whosonfirst/go-whosonfirst-pip/issues/15
				https://github.com/whosonfirst/go-whosonfirst-pip/issues/18
			*/

			if is_contained {
				mu.Lock()
				contained = append(contained, wof)
				mu.Unlock()
			}
		}

		go wg_ensure(wof)
	}

	wg.Wait()

	// All done checking the results

	d := time.Since(t)

	var tm metrics.Timer
	tm = *p.Metrics.TimeToContain
	go tm.Update(d)

	return contained, d
}

func (p WOFPointInPolygon) LoadGeoJSON(path string) (*geojson.WOFFeature, error) {

	t := time.Now()

	feature, err := geojson.UnmarshalFile(path)

	d := time.Since(t)

	var tm metrics.Timer
	tm = *p.Metrics.TimeToUnmarshal

	go tm.Update(d)

	if err != nil {
		p.Logger.Error("failed to unmarshal %s, because %s", path, err)
		return nil, err
	}

	var c metrics.Counter
	c = *p.Metrics.CountUnmarshal
	go c.Inc(1)

	return feature, err
}

func (p WOFPointInPolygon) LoadPolygons(wof *geojson.WOFSpatial) ([]*geojson.WOFPolygon, error) {

	id := wof.Id

	cache, ok := p.Cache.Get(id)

	if ok {

		var c metrics.Counter
		c = *p.Metrics.CountCacheHit
		go c.Inc(1)

		polygons := cache.([]*geojson.WOFPolygon)
		return polygons, nil
	}

	var c metrics.Counter
	c = *p.Metrics.CountCacheMiss
	go c.Inc(1)

	abs_path := utils.Id2AbsPath(p.Source, id)
	feature, err := p.LoadGeoJSON(abs_path)

	if err != nil {
		return nil, err
	}

	polygons, poly_err := p.LoadPolygonsForFeature(feature)

	if poly_err != nil {
		return nil, poly_err
	}

	return polygons, nil
}

func (p WOFPointInPolygon) LoadPolygonsForFeature(feature *geojson.WOFFeature) ([]*geojson.WOFPolygon, error) {

	id := feature.Id()

	polygons := feature.GeomToPolygons()
	var points int

	for _, pl := range polygons {
		points += pl.CountPoints()
	}

	p.Logger.Debug("%d has %d points", id, points)

	if points >= p.CacheTrigger {

		p.Logger.Debug("caching %d because it has E_EXCESSIVE_POINTS (%d)", id, points)

		var c metrics.Counter
		c = *p.Metrics.CountCacheSet

		evicted := p.Cache.Add(id, polygons)

		if evicted == true {

			cache_size := p.CacheSize
			cache_set := c.Count()

			p.Logger.Warning("starting to push thing out of the cache %d sets on a cache size of %d", cache_set, cache_size)
		}

		go c.Inc(1)
	}

	return polygons, nil
}

func (p WOFPointInPolygon) IsKnownPlacetype(pt string) bool {

	_, ok := p.Placetypes[pt]

	if ok {
		return true
	} else {
		return false
	}
}
