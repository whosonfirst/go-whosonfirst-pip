package pip

import (
	"github.com/dhconnelly/rtreego"
	"github.com/whosonfirst/go-whosonfirst-geojson"
	_ "github.com/kellydunn/golang-geo"
)

type WOFPointInPolygon struct {
	Rtree *rtreego.Rtree
}

func PointInPolygon() *WOFPointInPolygon {

	rt := rtreego.NewTree(2, 25, 50)

	return &WOFPointInPolygon{
		Rtree: rt,
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

	// fmt.Printf("%v\n", spatial.Bounds())

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

// maybe take a list of *geojson.WOFSpatial instead...

func (p WOFPointInPolygon) Contained (lat float64, lon float64, results []rtreego.Spatial) []rtreego.Spatial {

        contained := make([]rtreego.Spatial, 0)

	/*

	pt := geo.NewPoint(lat, lon)

	for _, r := range results {

	    // get ID
	    // get path
	    // open file
	    // read geometry in to a geo.Polygon
	    // see also: https://github.com/kellydunn/golang-geo/blob/master/polygon_test.go#L138
	    // poly := WUB WUB WUB

	    if poly.Contains(pt) {
	       contained = append(contained, r)
	    }

	}

	*/

	return contained
}