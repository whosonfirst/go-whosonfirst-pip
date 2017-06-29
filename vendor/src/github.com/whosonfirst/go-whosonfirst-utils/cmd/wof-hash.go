package main

// ./bin/wof-hash `./bin/wof-expand -root /usr/local/mapzen/whosonfirst-data/data/ 101736545`
// 18cb083ac494220e18da5a799b1b76ec

import (
	"flag"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-utils"
	"log"
)

func main() {

	var geom = flag.Bool("geom", false, "Only hash a feature's geometry")

	flag.Parse()
	args := flag.Args()

	for _, path := range args {

		var hash string
		var err error

		if *geom {
			hash, err = utils.HashGeomFromFile(path)
		} else {
			hash, err = utils.HashFile(path)
		}

		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(hash)
	}
}
