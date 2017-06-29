package main

/*

./bin/wof-api -async -paginated -geojson-ls -geojson-ls-output test.txt -timings -param api_key=mapzen-xxxxxx -param method=whosonfirst.places.search -param placetype=venue -param neighbourhood_id=85834637
2017/06/27 18:50:52 time to 'whosonfirst.places.search': 5.20128877s

./bin/wof-validate-geojsonls -stats ./test.txt
2017/06/27 19:05:26 ./test.txt 1022 records processed 60.291534ms

*/

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

type Validator struct {
	Strict  bool
	Path    string
	LineNum int
}

func (v *Validator) Validate(raw string, ch chan bool) {

	defer func() {
		ch <- true
	}()

	var stub interface{}

	dec := json.NewDecoder(strings.NewReader(raw))

	for {

		err := dec.Decode(&stub)

		if err == io.EOF {
			break
		}

		if err != nil {

			msg := fmt.Sprintf("Failed to parse JSON at %s line %d", v.Path, v.LineNum)

			if v.Strict {
				log.Fatal(msg)
			}

			log.Println(msg)
			break
		}
	}

}

func main() {

	var procs = flag.Int("processes", runtime.NumCPU()*2, "The number of concurrent processes to use")
	var strict = flag.Bool("strict", false, "Whether or not to trigger a fatal error when invalid JSON is encountered")
	var stats = flag.Bool("stats", false, "Be chatty, with counts and stuff")

	flag.Parse()

	ch := make(chan bool, *procs)

	for i := 0; i < *procs; i++ {
		ch <- true
	}

	for _, path := range flag.Args() {

		t1 := time.Now()

		fh, err := os.Open(path)

		if err != nil {
			log.Fatal(err)
		}

		scanner := bufio.NewScanner(fh)
		lineno := 0

		for scanner.Scan() {

			<-ch

			lineno += 1
			raw := scanner.Text()

			v := Validator{
				Strict:  *strict,
				Path:    path,
				LineNum: lineno,
			}

			// Please to be passing a success/fail channel around here and leave
			// decisions about fatal errors to the main loop...
			// (20170627/thisisaaronland)

			v.Validate(raw, ch)
		}

		t2 := time.Since(t1)

		if *stats {
			log.Printf("%s %d records processed in %v\n", path, lineno, t2)
		}
	}

	os.Exit(0)
}
