package main

import (
       "bufio"
	"encoding/json"
	"flag"
	"github.com/tidwall/gjson"
	"github.com/whosonfirst/go-whosonfirst-crawl"
	"github.com/whosonfirst/go-whosonfirst-uri"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"sync"
	"time"
)

func main() {

	outfile := flag.String("out", "", "Where to write records (default is STDOUT)")
	exclude_deprecated := flag.Bool("exclude-deprecated", false, "Exclude records that have been deprecated.")
	exclude_superseded := flag.Bool("exclude-superseded", false, "Exclude records that have been superseded.")
	timings := flag.Bool("timings", false, "Print timings")

	procs := flag.Int("processes", runtime.NumCPU()*2, "The number of concurrent processes to use")

	flag.Parse()

	runtime.GOMAXPROCS(*procs)

	var wr *bufio.Writer

	if *outfile != "" {

	   fh, err := os.Create(*outfile)

	   if err != nil {
	   	log.Fatal(err)
	   }

	   wr = bufio.NewWriter(fh)

	} else {
		wr = bufio.NewWriter(os.Stdout)
	}

	mu := new(sync.Mutex)

	for _, root := range flag.Args() {

		callback := func(path string, info os.FileInfo) error {

			if info.IsDir() {
				return nil
			}

			is_wof, err := uri.IsWOFFile(path)

			if err != nil {
				log.Printf("unable to determine whether %s is a WOF file, because %s\n", path, err)
				return err
			}

			if !is_wof {
				// log.Printf("%s is not a WOF file\n", path)
				return nil
			}

			is_alt, err := uri.IsAltFile(path)

			if err != nil {
				log.Printf("unable to determine whether %s is an alt (WOF) file, because %s\n", path, err)
				return err
			}

			if is_alt {
				// log.Printf("%s is an alt (WOF) file\n", path)
				return nil
			}

			fh, err := os.Open(path)

			if err != nil {
				log.Printf("failed to open %s, because %s\n", path, err)
				return err
			}

			defer fh.Close()

			body, err := ioutil.ReadAll(fh)

			if err != nil {
				log.Printf("failed to read %s, because %s\n", path, err)
				return err
			}

			var feature interface{}

			err = json.Unmarshal(body, &feature)

			if err != nil {
				log.Printf("failed to parse %s, because %s\n", path, err)
				return err
			}

			if *exclude_deprecated {

				rsp := gjson.GetBytes(body, "properties.edtf:deprecated")

				if rsp.Exists() {

					deprecated := rsp.String()

					if deprecated != "" && deprecated != "uuuu" {
						return nil
					}
				}
			}

			if *exclude_superseded {

				rsp := gjson.GetBytes(body, "properties.wof:superseded_by")

				if rsp.Exists() {

					superseded_by := rsp.Array()

					if len(superseded_by) > 0 {
						return nil
					}
				}
			}

			body, err = json.Marshal(feature)

			if err != nil {
				log.Printf("failed to parse %s, because %s\n", path, err)
				return err
			}

			mu.Lock()
			defer mu.Unlock()

			_, err = wr.Write(body)

			if err != nil {
				return err
			}

			wr.Write([]byte("\n"))
			wr.Flush()

			return nil
		}

		t1 := time.Now()
		
		c := crawl.NewCrawler(root)
		c.Crawl(callback)

		t2 := time.Since(t1)

		if *timings {
			log.Printf("time to process %s: %v\n", root, t2)
		}
	}
}
