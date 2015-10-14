package pip

import (
       "encoding/csv"
       "path"
       "io"
       "os"
       _ "fmt"
       "time"
	rtreego "github.com/dhconnelly/rtreego"
	geo "github.com/kellydunn/golang-geo"
	geojson "github.com/whosonfirst/go-whosonfirst-geojson"
	utils "github.com/whosonfirst/go-whosonfirst-utils"
)

type WOFPointInPolygonTiming struct {
     Event string
     Duration float64
}

type WOFPointInPolygon struct {
	Rtree  *rtreego.Rtree
	Source string
}

func PointInPolygon(source string) *WOFPointInPolygon {

	rt := rtreego.NewTree(2, 25, 50)

	return &WOFPointInPolygon{
		Rtree:  rt,
		Source: source,
	}
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
			continue
		}

		index_err := p.IndexGeoJSONFile(abs_path)

		if index_err != nil {
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

		// Go, you so wacky...
		// https://golang.org/doc/effective_go.html#interface_conversions

		wof := r.(*geojson.WOFSpatial)
		inflated = append(inflated, wof)
	}

	return inflated
}

func (p WOFPointInPolygon) GetByLatLon(lat float64, lon float64) ([]*geojson.WOFSpatial, []*WOFPointInPolygonTiming) {

     	timings := make([]*WOFPointInPolygonTiming, 0)

	t1a := time.Now()

	intersects := p.GetIntersectsByLatLon(lat, lon)

	t1b := float64(time.Since(t1a)) / 1e9
	timings = append(timings, &WOFPointInPolygonTiming{"intersects", t1b})

	t2a := time.Now()

	inflated := p.InflateSpatialResults(intersects)

	t2b := float64(time.Since(t2a)) / 1e9
	timings = append(timings, &WOFPointInPolygonTiming{"inflate", t2b})

	t3a := time.Now()

	contained := p.EnsureContained(lat, lon, inflated)

	t3b := float64(time.Since(t3a)) / 1e9
	timings = append(timings, &WOFPointInPolygonTiming{"contained", t3b})

	return contained, timings
}

func (p WOFPointInPolygon) GetByLatLonForPlacetype(lat float64, lon float64, placetype string) ([]*geojson.WOFSpatial, []*WOFPointInPolygonTiming) {

	possible, timings := p.GetByLatLon(lat, lon)

	t1a := time.Now()

	filtered := p.FilterByPlacetype(possible, placetype)

	t1b := float64(time.Since(t1a)) / 1e9
	timings = append(timings, &WOFPointInPolygonTiming{"placetype", t1b})

	return filtered, timings
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

		// please cache me... somewhere... somehow...
		// (20151013/thisisaaronland)

		id := wof.Id
		path := utils.Id2AbsPath(p.Source, id)

		feature, err := geojson.UnmarshalFile(path)

		if err != nil {
			// please log me
			continue
		}

		// basically return this from the cache (for wof.Id)
		// (20151013/thisisaaronland)

		// it might also be nice to be able to return this as
		// an iterator to save build large polygons but today
		// we'll just assume that is yak-shaving on move along
		// (20151013/thisisaaronland)

		polygons := feature.GeomToPolygons()

		is_contained := false

		for _, poly := range polygons {

			if poly.Contains(pt) {
				is_contained = true
				break
			}
		}

		if is_contained {
			contained = append(contained, wof)
		}

	}

	return contained
}
