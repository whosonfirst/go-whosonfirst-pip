package main

import (
	"flag"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-pip"
)

func main() {

	flag.Parse()
	args := flag.Args()

	p := pip.PointInPolygon()

	for _, path := range args {
		fmt.Println(path)
		p.ImportFile(path)
	}

	fmt.Println(p.Rtree.Size())
}
