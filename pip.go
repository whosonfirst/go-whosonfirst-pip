package pip

import (
	rtreego "github.com/dhconnelly/rtreego"
	lru "github.com/hashicorp/golang-lru"
	geo "github.com/kellydunn/golang-geo"
	metrics "github.com/rcrowley/go-metrics"
	csv "github.com/whosonfirst/go-whosonfirst-csv"
	geojson "github.com/whosonfirst/go-whosonfirst-geojson"
	log "github.com/whosonfirst/go-whosonfirst-log"
	utils "github.com/whosonfirst/go-whosonfirst-utils"
	"io"
	golog "log"
	"os"
	"path"
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

	logger := log.NewWOFLogger(os.Stdout, "[pip-server]", "debug")

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

	if format == "log" {
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

	feature, parse_err := p.LoadGeoJSON(path)

	if parse_err != nil {
		return parse_err
	}

	return p.IndexGeoJSONFeature(feature)
}

func (p WOFPointInPolygon) IndexGeoJSONFeature(feature *geojson.WOFFeature) error {

	spatial, spatial_err := feature.EnSpatialize()

	if spatial_err != nil {
		p.Logger.Error("failed to enspatialize feature, because %s", spatial_err)
		return spatial_err
	}

	pt := spatial.Placetype

	_, ok := p.Placetypes[pt]

	if ok {
		p.Placetypes[pt] += 1
	} else {
		p.Placetypes[pt] = 1
	}

	p.Rtree.Insert(spatial)

	go p.LoadPolygonsForFeature(feature)
	return nil
}

func (p WOFPointInPolygon) IndexMetaFile(csv_file string) error {

	reader, reader_err := csv.NewDictReader(csv_file)

	if reader_err != nil {
		p.Logger.Error("failed to create CSV reader , because %s", reader_err)
		return reader_err
	}

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

	t := time.Now()

	pt := rtreego.Point{lon, lat}
	bbox, _ := rtreego.NewRect(pt, []float64{0.0001, 0.0001}) // how small can I make this?

	results := p.Rtree.SearchIntersect(bbox)

	d := time.Since(t)

	var tm metrics.Timer
	tm = *p.Metrics.TimeToIntersect
	tm.Update(d)

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
	tm.Update(d)

	return inflated, d
}

func (p WOFPointInPolygon) GetByLatLon(lat float64, lon float64) ([]*geojson.WOFSpatial, []*WOFPointInPolygonTiming) {

	// See that: placetype == ""; see below for details
	return p.GetByLatLonForPlacetype(lat, lon, "")
}

func (p WOFPointInPolygon) GetByLatLonForPlacetype(lat float64, lon float64, placetype string) ([]*geojson.WOFSpatial, []*WOFPointInPolygonTiming) {

	var c metrics.Counter
	c = *p.Metrics.CountLookups
	c.Inc(1)

	t := time.Now()

	timings := make([]*WOFPointInPolygonTiming, 0)

	intersects, duration := p.GetIntersectsByLatLon(lat, lon)
	timings = append(timings, NewWOFPointInPolygonTiming("intersects", duration))

	inflated, duration := p.InflateSpatialResults(intersects)
	timings = append(timings, NewWOFPointInPolygonTiming("inflate", duration))

	// See what's going on here? We are filtering by placetype before
	// do a final point-in-poly lookup so we don't try to load country
	// records while only searching for localities

	filtered := make([]*geojson.WOFSpatial, 0)

	if placetype != "" {
		filtered, duration = p.FilterByPlacetype(inflated, placetype)
		timings = append(timings, NewWOFPointInPolygonTiming("filter", duration))
	} else {
		filtered = inflated
	}

	contained, duration := p.EnsureContained(lat, lon, filtered)
	timings = append(timings, NewWOFPointInPolygonTiming("contain", duration))

	d := time.Since(t)

	var tm metrics.Timer
	tm = *p.Metrics.TimeToProcess
	tm.Update(d)

	ttp := float64(d) / 1e9

	if ttp > 0.5 {
		p.Logger.Warning("time to process %f,%f (%s) exceeds 0.5 seconds: %f", lat, lon, placetype, ttp)
	}

	return contained, timings
}

func (p WOFPointInPolygon) FilterByPlacetype(results []*geojson.WOFSpatial, placetype string) ([]*geojson.WOFSpatial, time.Duration) {

	t := time.Now()

	filtered := make([]*geojson.WOFSpatial, 0)

	for _, r := range results {
		if r.Placetype == placetype {
			filtered = append(filtered, r)
		}
	}

	d := time.Since(t)

	return filtered, d
}

func (p WOFPointInPolygon) EnsureContained(lat float64, lon float64, results []*geojson.WOFSpatial) ([]*geojson.WOFSpatial, time.Duration) {

	t := time.Now()

	contained := make([]*geojson.WOFSpatial, 0)

	pt := geo.NewPoint(lat, lon)

	// please do this with a waitgroup or something

	for _, wof := range results {

		polygons, err := p.LoadPolygons(wof)

		if err != nil {
			// please log me
			continue
		}

		is_contained := false

		count := len(polygons)
		iters := 0

		for _, poly := range polygons {

			iters += 1

			if poly.Contains(pt) {
				p.Logger.Debug("point is contained after checking %d/%d polygons", iters, count)
				is_contained = true
				break
			}

		}

		if is_contained {
			contained = append(contained, wof)
		}

	}

	d := time.Since(t)

	var tm metrics.Timer
	tm = *p.Metrics.TimeToContain
	tm.Update(d)

	count_in := len(results)
	count_out := len(contained)

	p.Logger.Debug("contained: %d/%d\n", count_out, count_in)

	return contained, d
}

func (p WOFPointInPolygon) LoadGeoJSON(path string) (*geojson.WOFFeature, error) {

	t := time.Now()

	feature, err := geojson.UnmarshalFile(path)

	d := time.Since(t)

	ttl := float64(d) / 1e9

	if ttl > 0.1 {
		p.Logger.Warning("time to load %s exceeds 0.1 seconds: %f", path, ttl)
	}

	var tm metrics.Timer
	tm = *p.Metrics.TimeToUnmarshal

	tm.Update(d)

	if err != nil {
		p.Logger.Error("failed to unmarshal %s, because %s", path, err)
		return nil, err
	}

	var c metrics.Counter
	c = *p.Metrics.CountUnmarshal
	c.Inc(1)

	return feature, err
}

func (p WOFPointInPolygon) LoadPolygons(wof *geojson.WOFSpatial) ([]*geo.Polygon, error) {

	id := wof.Id

	cache, ok := p.Cache.Get(id)

	if ok {

		var c metrics.Counter
		c = *p.Metrics.CountCacheHit
		c.Inc(1)

		polygons := cache.([]*geo.Polygon)
		return polygons, nil
	}

	var c metrics.Counter
	c = *p.Metrics.CountCacheMiss
	c.Inc(1)

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

func (p WOFPointInPolygon) LoadPolygonsForFeature(feature *geojson.WOFFeature) ([]*geo.Polygon, error) {

	polygons := feature.GeomToPolygons()
	var points int

	for _, p := range polygons {
		points += len(p.Points())
	}

	if points >= p.CacheTrigger {

		id := feature.WOFId()

		p.Logger.Info("caching %d because it has E_EXCESSIVE_POINTS (%d)", id, points)

		var c metrics.Counter
		c = *p.Metrics.CountCacheSet

		evicted := p.Cache.Add(id, polygons)

		if evicted == true {

			cache_size := p.CacheSize
			cache_set := c.Count()

			p.Logger.Warning("starting to push thing out of the cache %d sets on a cache size of %d", cache_set, cache_size)
		}

		c.Inc(1)
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
