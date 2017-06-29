package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-uri"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type FeatureCollection struct {
	Type     string        `json:"type"`
	Features []interface{} `json:"features"`
}

func main() {

	var repo = flag.String("repo", "", "The path to the repository where the files you're passing to d2fc are stored.")
	flag.Parse()

	collection := make([]interface{}, 0)

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {

		ln := scanner.Text()

		rel_path := strings.TrimSpace(ln)
		abs_path := filepath.Join(*repo, rel_path)

		ok, err := uri.IsWOFFile(abs_path)

		if err != nil {
			msg := fmt.Sprintf("Failed to determine if %s is a WOF file, becauuse %s", abs_path, err.Error())
			log.Fatal(msg)
		}

		if !ok {
			msg := fmt.Sprintf("%s is not a WOF file", abs_path)
			log.Fatal(msg)
		}

		body, err := ioutil.ReadFile(abs_path)

		if err != nil {
			msg := fmt.Sprintf("Failed to read %s, becauuse %s", abs_path, err.Error())
			log.Fatal(msg)
		}

		var feature interface{}
		err = json.Unmarshal(body, &feature)

		if err != nil {
			msg := fmt.Sprintf("Failed to unmarshal %s, becauuse %s", abs_path, err.Error())
			log.Fatal(msg)
		}

		collection = append(collection, feature)
	}

	fc := FeatureCollection{
		Type:     "FeatureCollection",
		Features: collection,
	}

	b, err := json.Marshal(fc)

	if err != nil {
		msg := fmt.Sprintf("Failed to marshal feature collection, becauuse %s", err.Error())
		log.Fatal(msg)
	}

	writers := []io.Writer{
		os.Stdout,
	}

	multi := io.MultiWriter(writers...)
	writer := bufio.NewWriter(multi)

	writer.Write(b)
	writer.Flush()
}
