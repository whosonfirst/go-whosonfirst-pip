package main

// ./bin/wof-hash `./bin/wof-expand -root /usr/local/mapzen/whosonfirst-data/data/ 101736545`
// 18cb083ac494220e18da5a799b1b76ec

import (
	"flag"
	"fmt"
	utils "github.com/whosonfirst/go-whosonfirst-utils"
)

func main() {

	flag.Parse()
	args := flag.Args()

	for _, path := range args {
		hash, _ := utils.HashFile(path)
		fmt.Println(hash)
	}
}
