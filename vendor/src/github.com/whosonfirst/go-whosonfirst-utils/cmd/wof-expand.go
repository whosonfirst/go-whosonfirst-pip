package main

import (
	"flag"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-uri"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {

	var root = flag.String("root", "", "The directory where Who's On First records are stored. If empty defaults to the current working directory + \"/data\".")
	var prefix = flag.String("prefix", "", "Prepend this prefix to all paths")

	var alt = flag.Bool("alternate", false, "Encode URI as an alternate geometry")
	var strict = flag.Bool("strict", false, "Ensure that the source for an alternate geometry is valid (see also: go-whosonfirst-sources)")
	var source = flag.String("source", "", "The source of the alternate geometry")
	var function = flag.String("function", "", "The function of the alternate geometry (optional)")
	var extras = flag.String("extras", "", "A comma-separated list of extra information to include with an alternate geometry (optional)")

	flag.Parse()

	if *root == "" {

		cwd, err := os.Getwd()

		if err != nil {
			log.Fatal(err)
		}

		*root = filepath.Join(cwd, "data")
	}

	for _, str_id := range flag.Args() {

		id, err := strconv.Atoi(str_id)

		if err != nil {
			log.Fatal("Unable to parse %s, because %v", str_id, err)
		}

		var args *uri.URIArgs

		if *alt {

			parsed := make([]string, 0)

			for _, e := range strings.Split(*extras, ",") {

				e = strings.Trim(e, " ")

				if e != "" {
					parsed = append(parsed, e)
				}
			}

			args = uri.NewAlternateURIArgs(*source, *function, parsed...)
			args.Strict = *strict

		} else {
			args = uri.NewDefaultURIArgs()
		}

		wof_path, err := uri.Id2RelPath(id, args)

		if err != nil {
			log.Printf("failed to generate a URI for %s, because '%v'\n", str_id, err)
			continue
		}

		if *prefix != "" {
			wof_path = path.Join(*prefix, wof_path)
		}

		if *root != "" {
			wof_path = path.Join(*root, wof_path)
		}

		fmt.Println(wof_path)
	}
}
