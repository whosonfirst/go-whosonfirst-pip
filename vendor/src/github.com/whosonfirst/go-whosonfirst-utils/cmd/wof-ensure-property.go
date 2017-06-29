package main

import (
	"flag"
	"fmt"
	"github.com/tidwall/gjson"
	"github.com/whosonfirst/go-whosonfirst-crawl"
	"github.com/whosonfirst/go-whosonfirst-csv"
	"github.com/whosonfirst/go-whosonfirst-uri"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

func main() {

	repo := flag.String("repo", "/usr/local/data/whosonfirst-data", "The WOF repo whose files you want to test.")
	procs := flag.Int("processes", runtime.NumCPU()*2, "The number of concurrent processes to use")
	prop := flag.String("property", "", "The dotted notation for the property whose existence you want to test.")

	flag.Parse()

	runtime.GOMAXPROCS(*procs)

	_, err := os.Stat(*repo)

	if os.IsNotExist(err) {
		log.Fatal(err)
	}

	data := filepath.Join(*repo, "data")

	_, err = os.Stat(data)

	if os.IsNotExist(err) {
		log.Fatal(err)
	}

	if *prop == "" {
		log.Fatal("You forgot to specify anything to check for.")
	}

	fieldnames := []string{"id", "path", "details"}
	writer, err := csv.NewDictWriter(os.Stdout, fieldnames)

	writer.WriteHeader()

	mu := new(sync.Mutex)

	callback := func(path string, info os.FileInfo) error {

		if info.IsDir() {
			return nil
		}

		is_wof, err := uri.IsWOFFile(path)

		if err != nil {
			return err
		}

		if !is_wof {
			return nil
		}

		is_alt, err := uri.IsAltFile(path)

		if err != nil {
			return err
		}

		if is_alt {
			return nil
		}

		fh, err := os.Open(path)
		defer fh.Close()

		if err != nil {
			return err
		}

		body, err := ioutil.ReadAll(fh)

		if err != nil {
			return err
		}

		var jpath string

		if strings.HasPrefix(*prop, "properties") {
			jpath = *prop
		} else {
			jpath = fmt.Sprintf("properties.%s", *prop)
		}

		result := gjson.GetBytes(body, jpath)

		if result.Exists() {
			return nil
		}

		id, err := uri.IdFromPath(path)

		if err != nil {
			return err
		}

		str_id := strconv.FormatInt(id, 10)

		details := fmt.Sprintf("missing '%s'", jpath)

		mu.Lock()
		defer mu.Unlock()

		row := make(map[string]string)
		row["id"] = str_id
		row["path"] = path
		row["details"] = details

		writer.WriteRow(row)
		return nil
	}

	cr := crawl.NewCrawler(data)
	cr.Crawl(callback)
}
