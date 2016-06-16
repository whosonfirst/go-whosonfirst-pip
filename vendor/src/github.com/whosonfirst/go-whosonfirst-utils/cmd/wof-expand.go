package main

import (
	"flag"
	"fmt"
	utils "github.com/whosonfirst/go-whosonfirst-utils"
	"log"
	"path"
	"strconv"
)

func main() {

	var root = flag.String("root", "", "A root directory for absolute paths")
	var prefix = flag.String("prefix", "", "Prepend this prefix to all paths")

	flag.Parse()

	for _, str_id := range flag.Args() {
		id, err := strconv.Atoi(str_id)

		if err != nil {
			log.Fatal("Unable to parse %s, because %v", str_id, err)
		}

		wof_path := utils.Id2RelPath(id)

		if *prefix != "" {
			wof_path = path.Join(*prefix, wof_path)
		}

		if *root != "" {
			wof_path = path.Join(*root, wof_path)
		}

		fmt.Println(wof_path)
	}
}
