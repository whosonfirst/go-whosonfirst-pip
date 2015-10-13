package pip

import (
	"github.com/dhconnelly/rtreego"
	"github.com/whosonfirst/go-whosonfirst-geojson"
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

	p.Rtree.Insert(spatial)
	return nil
}

func (p WOFPointInPolygon) GetByLatLon(lat float64, lon float64) []rtreego.Spatial{

	pt := rtreego.Point{lon, lat}
	bbox, _ := rtreego.NewRect(pt, []float64{0.0001, 0.0001})	// how small can I make this?

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