package pip

import (
	"github.com/dhconnelly/rtreego"
	"github.com/whosonfirst/go-whosonfirst-geojson"
	"github.com/whosonfirst/go-whosonfirst-utils"
	"github.com/kellydunn/golang-geo"
)

type WOFPointInPolygon struct {
	Rtree *rtreego.Rtree
	Source string
}

func PointInPolygon(source string) *WOFPointInPolygon {

	rt := rtreego.NewTree(2, 25, 50)

	return &WOFPointInPolygon{
		Rtree: rt,
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

func (p WOFPointInPolygon) GetByLatLon(lat float64, lon float64) []rtreego.Spatial {

	pt := rtreego.Point{lon, lat}
	bbox, _ := rtreego.NewRect(pt, []float64{0.0001, 0.0001}) // how small can I make this?

	results := p.Rtree.SearchIntersect(bbox)
	return results
}

func (p WOFPointInPolygon) InflateResults(results []rtreego.Spatial) []*geojson.WOFSpatial {

	inflated := make([]*geojson.WOFSpatial, 0)

	for _, r := range results {

		// https://golang.org/doc/effective_go.html#interface_conversions
		wof := r.(*geojson.WOFSpatial)
		inflated = append(inflated, wof)
	}

	return inflated
}

func (p WOFPointInPolygon) FilterByPlacetype(results []*geojson.WOFSpatial, placetype string) []*geojson.WOFSpatial {

	filtered := make([]*geojson.WOFSpatial, 0)

	for _, r := range results {
	        if (r.Placetype == placetype){
		   filtered = append(filtered, r)
		}   
	}

	return filtered
}

func (p WOFPointInPolygon) Contained (lat float64, lon float64, results []*geojson.WOFSpatial) []*geojson.WOFSpatial {

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