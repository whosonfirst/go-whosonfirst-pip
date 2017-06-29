package main

import (
	"flag"
	"github.com/whosonfirst/go-whosonfirst-uri"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {

	var root = flag.String("root", "", "The directory where Who's On First records are stored. If empty defaults to the current working directory + \"/data\".")
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

	ids := flag.Args()

	for _, str_id := range ids {

		// id, err := strconv.ParseInt(str_id, 10, 64)
		id, err := strconv.Atoi(str_id)

		abs_path, err := uri.Id2AbsPath(*root, id, args)

		if err != nil {
			log.Fatal(err)
		}

		fh, err := os.Open(abs_path)

		if err != nil {
			log.Fatal(err)
		}

		_, err = io.Copy(os.Stdout, fh)
	}
}
