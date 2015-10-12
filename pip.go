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

func (wof WOFPointInPolygon) ImportFile(source string) error {

	feature, parse_err := geojson.UnmarshalFile(source)

	if parse_err != nil {
		return parse_err
	}

	bounds, bounds_err := feature.Bounds()

	if bounds_err != nil {
		return bounds_err
	}

	wof.Rtree.Insert(bounds)
	return nil
}

/*
func (wof WOFPointInPolygon) LookupPoint(lat float64, lon float64) {

	q := rtreego.Point{lat, lon}
	k := 5

	results = wof.Rtree.SearchNearestNeighbors(q, k)
}
*/