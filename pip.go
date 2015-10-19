package pip

import (
	"encoding/csv"
	"fmt"
	rtreego "github.com/dhconnelly/rtreego"
	lru "github.com/hashicorp/golang-lru"
	geo "github.com/kellydunn/golang-geo"
	metrics "github.com/rcrowley/go-metrics"
	geojson "github.com/whosonfirst/go-whosonfirst-geojson"
	utils "github.com/whosonfirst/go-whosonfirst-utils"
	"io"
	_ "log"
	"os"
	"path"
	"time"
)

type WOFPointInPolygonTiming struct {
	Event    string
	Duration float64
}

func NewWOFPointInPolygonTiming(event string, d time.Duration) *WOFPointInPolygonTiming {

	df := float64(d) / 1e9

	t := WOFPointInPolygonTiming{Event: event, Duration: df}
	return &t

}

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

	registry.Register("lookups", cnt_lookups)
	registry.Register("unmarshaled", cnt_unmarshal)
	registry.Register("cache-hit", cnt_cache_hit)
	registry.Register("cache-miss", cnt_cache_miss)
	registry.Register("cache-set", cnt_cache_set)
	registry.Register("time-to-process", tm_process)
	registry.Register("time-to-unmarshal", tm_unmarshal)
	registry.Register("time-to-intersect", tm_intersect)
	// registry.Register("time-to-inflate", tm_inflate)
	registry.Register("time-to-contain", tm_contain)

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

	return &m
}

type WOFPointInPolygon struct {
	Rtree      *rtreego.Rtree
	Cache      *lru.Cache
	CacheSize  int
	Source     string
	Placetypes map[string]int
	Metrics    *WOFPointInPolygonMetrics
}

func NewPointInPolygon(source string, cache_size int) (*WOFPointInPolygon, error) {

	rt := rtreego.NewTree(2, 25, 50)

	cache, err := lru.New(cache_size)

	if err != nil {
		return nil, err
	}

	m := NewPointInPolygonMetrics()

	pt := make(map[string]int)

	pip := WOFPointInPolygon{
		Rtree:      rt,
		Source:     source,
		Cache:      cache,
		CacheSize:  cache_size,
		Placetypes: pt,
		Metrics:    m,
	}

	return &pip, nil
}

func (p WOFPointInPolygon) IndexGeoJSONFile(path string) error {

	feature, parse_err := p.LoadGeoJSON(path)

	if parse_err != nil {
		return parse_err
	}

	return p.IndexGeoJSONFeature(feature)
}

func (p WOFPointInPolygon) IndexGeoJSONFeature(feature *geojson.WOFFeature) error {

	spatial, spatial_err := feature.EnSpatialize()

	if spatial_err != nil {
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

	p.LoadPolygonsForFeature(feature)
	return nil
}

func (p WOFPointInPolygon) IndexMetaFile(csv_file string, offset int) error {

	body, read_err := os.Open(csv_file)

	if read_err != nil {
		return read_err
	}

	r := csv.NewReader(body)

	for {
		record, err := r.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		// sudo my kingdom for a DictReader in Go...
		// (20151013/thisisaaronland)

		rel_path := record[offset]
		abs_path := path.Join(p.Source, rel_path)

		_, err = os.Stat(abs_path)

		if os.IsNotExist(err) {
			// fmt.Printf("OH NO - can't find %s\n", abs_path)
			continue
		}

		index_err := p.IndexGeoJSONFile(abs_path)

		if index_err != nil {
			// fmt.Printf("FAILED TO INDEX %s, because %s", abs_path, index_err)
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

	for _, wof := range results {

		// t1a := time.Now()

		polygons, err := p.LoadPolygons(wof)

		if err != nil {
			// please log me
			continue
		}

		// t1b := float64(time.Since(t1a)) / 1e9
		// fmt.Printf("[debug] time to load polygons is %f\n", t1b)

		is_contained := false

		// count := len(polygons)
		iters := 0

		// t3a := time.Now()

		for _, poly := range polygons {

			iters += 1

			if poly.Contains(pt) {
				is_contained = true
				break
			}

		}

		// t3b := float64(time.Since(t3a)) / 1e9
		// fmt.Printf("[debug] time to check containment (%t) after %d/%d possible iterations is %f\n", is_contained, iters, count, t3b)

		if is_contained {
			contained = append(contained, wof)
		}

	}

	d := time.Since(t)

	var tm metrics.Timer
	tm = *p.Metrics.TimeToContain
	tm.Update(d)

	// count_in := len(results)
	// count_out := len(contained)

	// fmt.Printf("[debug] contained: %d/%d\n", count_out, count_in)
	return contained, d
}

func (p WOFPointInPolygon) LoadGeoJSON(path string) (*geojson.WOFFeature, error) {

	t := time.Now()

	feature, err := geojson.UnmarshalFile(path)

	d := time.Since(t)

	var tm metrics.Timer
	tm = *p.Metrics.TimeToUnmarshal

	tm.Update(d)

	if err != nil {
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

	// t3a := time.Now()

	polygons := feature.GeomToPolygons()
	var points int

	for _, p := range polygons {
		points += len(p.Points())
	}

	// t3b := float64(time.Since(t3a)) / 1e9
	// fmt.Printf("[debug] time to convert geom to polygons (%d points) is %f\n", points, t3b)

	if points >= 1000 {

		var c metrics.Counter
		c = *p.Metrics.CountCacheSet

		id := feature.WOFId()
		evicted := p.Cache.Add(id, polygons)

		if evicted == true {

			cache_size := p.CacheSize
			cache_set := c.Count()

			fmt.Printf("starting to push thing out of the cache %d sets on a cache size of %d", cache_set, cache_size)
		}

		c.Inc(1)

		// fmt.Printf("[cache] %d because so many points (%d)\n", id, points)
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
