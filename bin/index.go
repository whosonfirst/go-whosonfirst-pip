package main

import (
	"flag"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-pip"
)

func main() {

	flag.Parse()
	args := flag.Args()

	// pip := PointInPolygon()

	for _, path := range args {
		//pip.ImportFile(path)
		fmt.Println(path)
	}
}
