package pip

import (
	"encoding/csv"
	"fmt"
	rtreego "github.com/dhconnelly/rtreego"
	lru "github.com/hashicorp/golang-lru"
	geo "github.com/kellydunn/golang-geo"
	geojson "github.com/whosonfirst/go-whosonfirst-geojson"
	utils "github.com/whosonfirst/go-whosonfirst-utils"
	"io"
	"os"
	"path"
	"time"
)

type WOFPointInPolygonTiming struct {
	Event    string
	Duration float64
}

type WOFPointInPolygon struct {
	Rtree      *rtreego.Rtree
	Cache      *lru.Cache
	Source     string
	Placetypes map[string]int
}

func PointInPolygon(source string) (*WOFPointInPolygon, error) {

	rt := rtreego.NewTree(2, 25, 50)

	cache_size := 256
	cache, err := lru.New(cache_size)

	if err != nil {
		return nil, err
	}

	pt := make(map[string]int)

	pip := WOFPointInPolygon{
		Rtree:      rt,
		Source:     source,
		Cache:      cache,
		Placetypes: pt,
	}

	return &pip, nil
}

func (p WOFPointInPolygon) IndexGeoJSONFile(source string) error {

	feature, parse_err := geojson.UnmarshalFile(source)

	if parse_err != nil {
		return parse_err
	}

	return p.IndexGeoJSONFeature(*feature)
}

func (p WOFPointInPolygon) IndexGeoJSONFeature(feature geojson.WOFFeature) error {

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

func (p WOFPointInPolygon) GetIntersectsByLatLon(lat float64, lon float64) []rtreego.Spatial {

	pt := rtreego.Point{lon, lat}
	bbox, _ := rtreego.NewRect(pt, []float64{0.0001, 0.0001}) // how small can I make this?

	results := p.Rtree.SearchIntersect(bbox)
	return results
}

// maybe just merge this above - still unsure (20151013/thisisaaronland)

func (p WOFPointInPolygon) InflateSpatialResults(results []rtreego.Spatial) []*geojson.WOFSpatial {

	inflated := make([]*geojson.WOFSpatial, 0)

	for _, r := range results {

		// https://golang.org/doc/effective_go.html#interface_conversions

		wof := r.(*geojson.WOFSpatial)
		inflated = append(inflated, wof)
	}

	return inflated
}

func (p WOFPointInPolygon) GetByLatLon(lat float64, lon float64) ([]*geojson.WOFSpatial, []*WOFPointInPolygonTiming) {

	// See that: placetype == ""; see below for details
	return p.GetByLatLonForPlacetype(lat, lon, "")
}

func (p WOFPointInPolygon) GetByLatLonForPlacetype(lat float64, lon float64, placetype string) ([]*geojson.WOFSpatial, []*WOFPointInPolygonTiming) {

	timings := make([]*WOFPointInPolygonTiming, 0)

	t1a := time.Now()

	intersects := p.GetIntersectsByLatLon(lat, lon)

	t1b := float64(time.Since(t1a)) / 1e9
	timings = append(timings, &WOFPointInPolygonTiming{"intersects", t1b})

	t2a := time.Now()

	inflated := p.InflateSpatialResults(intersects)

	t2b := float64(time.Since(t2a)) / 1e9
	timings = append(timings, &WOFPointInPolygonTiming{"inflate", t2b})

	// See what's going on here? We are filtering by placetype before
	// do a final point-in-poly lookup so we don't try to load country
	// records while only searching for localities

	filtered := make([]*geojson.WOFSpatial, 0)

	if placetype != "" {
		t3a := time.Now()

		filtered = p.FilterByPlacetype(inflated, placetype)

		t3b := float64(time.Since(t3a)) / 1e9
		timings = append(timings, &WOFPointInPolygonTiming{"placetype", t3b})
	} else {
		filtered = inflated
	}

	t4a := time.Now()

	contained := p.EnsureContained(lat, lon, filtered)

	t4b := float64(time.Since(t4a)) / 1e9
	timings = append(timings, &WOFPointInPolygonTiming{"contained", t4b})

	return contained, timings
}

func (p WOFPointInPolygon) FilterByPlacetype(results []*geojson.WOFSpatial, placetype string) []*geojson.WOFSpatial {

	filtered := make([]*geojson.WOFSpatial, 0)

	for _, r := range results {
		if r.Placetype == placetype {
			filtered = append(filtered, r)
		}
	}

	return filtered
}

func (p WOFPointInPolygon) EnsureContained(lat float64, lon float64, results []*geojson.WOFSpatial) []*geojson.WOFSpatial {

	contained := make([]*geojson.WOFSpatial, 0)

	pt := geo.NewPoint(lat, lon)

	for _, wof := range results {

		t1a := time.Now()

		polygons, err := p.LoadPolygons(wof)

		if err != nil {
			// please log me
			continue
		}

		t1b := float64(time.Since(t1a)) / 1e9
		fmt.Printf("[debug] time to load polygons is %f\n", t1b)

		is_contained := false

		count := len(polygons)
		iters := 0

		t3a := time.Now()

		for _, poly := range polygons {

			iters += 1

			if poly.Contains(pt) {
				is_contained = true
				break
			}

		}

		t3b := float64(time.Since(t3a)) / 1e9
		fmt.Printf("[debug] time to check containment (%t) after %d/%d possible iterations is %f\n", is_contained, iters, count, t3b)

		if is_contained {
			contained = append(contained, wof)
		}

	}

	count_in := len(results)
	count_out := len(contained)

	fmt.Printf("[debug] contained: %d/%d\n", count_out, count_in)
	return contained
}

func (p WOFPointInPolygon) LoadPolygons(wof *geojson.WOFSpatial) ([]*geo.Polygon, error) {

	id := wof.Id

	cache, ok := p.Cache.Get(id)

	if ok {

		fmt.Printf("[debug] return polygons from cache for %d\n", id)

		polygons := cache.([]*geo.Polygon)
		return polygons, nil
	}

	t2a := time.Now()

	abs_path := utils.Id2AbsPath(p.Source, id)
	feature, err := geojson.UnmarshalFile(abs_path)

	t2b := float64(time.Since(t2a)) / 1e9
	fmt.Printf("[debug] time to marshal %s is %f\n", abs_path, t2b)

	if err != nil {
		return nil, err
	}

	t3a := time.Now()

	polygons := feature.GeomToPolygons()
	var points int

	for _, p := range polygons {
		points += len(p.Points())
	}

	t3b := float64(time.Since(t3a)) / 1e9
	fmt.Printf("[debug] time to convert geom to polygons (%d points) is %f\n", points, t3b)

	if points >= 100 {

		if p.Cache.Len() == 256 { // PLEASE DO NOT HARDCODE ME...
			p.Cache.RemoveOldest()
		}

		_ = p.Cache.Add(id, polygons)
		fmt.Printf("[cache] %d because so many points (%d)\n", id, points)
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
