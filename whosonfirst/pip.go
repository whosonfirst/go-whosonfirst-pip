package pip

import (
       "github.com/whosonfirst/whosonfirst/geojson",
       "github.com/dhconnelly/rtreego"
)

type WOFPointInPolygon struct {
     Rtree *rtree.Rtree
}

func PointInPolygon(source string) *WOFPointInPolygon {

     rt := rtree.Rtree()    

     return &{ Rtree: rt }
}

func (wof WOFPointInPolygon) ImportFile(source string) {

     f := geojson.UnmarshalFile(source)
     wof.Rtree.Insert()
}

func (wof WOFPointInPolygon) LookupPoint(lat float64, lon float64) {

     q := rtreego.Point{lat, lon}
     k := 5

     results = wof.Rtree.SearchNearestNeighbors(q, k)
}