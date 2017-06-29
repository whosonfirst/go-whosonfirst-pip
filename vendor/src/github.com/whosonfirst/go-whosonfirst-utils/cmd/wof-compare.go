package main

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-csv"
	"github.com/whosonfirst/go-whosonfirst-uri"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

type WOFId struct {
	Id   int64
	Repo string
}

func NewWOFId(id int64, repo string) *WOFId {

	if repo == "" {
		repo = "whosonfirst-data"
	}

	w := WOFId{
		Id:   id,
		Repo: repo,
	}

	return &w
}

func HashWOFId(w *WOFId, sources map[string]string) (map[string]string, error) {

	hashes := make(map[string]string)

	wofid := w.Id

	rel_path, err := uri.Id2RelPath(int(wofid)) // OH GOD FIX ME...

	if err != nil {
		return hashes, err
	}

	wg := new(sync.WaitGroup)
	mu := new(sync.Mutex)

	for src, root := range sources {

		if src == "github" {
			root = strings.Replace(root, ":REPO:", w.Repo, -1)
		}

		wg.Add(1)

		go func(src string, root string, rel_path string, wg *sync.WaitGroup, mu *sync.Mutex) {

			defer wg.Done()

			hash, err := HashRecord(root, rel_path)

			mu.Lock()

			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to hash WOF ID %s%s because %s\n", root, rel_path, err)
				hashes[src] = ""
			} else {
				hashes[src] = hash
			}

			mu.Unlock()

		}(src, root, rel_path, wg, mu)
	}

	wg.Wait()

	return hashes, nil
}

func HashRecord(root string, rel_path string) (string, error) {

	uri := root + rel_path

	rsp, err := http.Get(uri)

	if err != nil {
		return "", err
	}

	defer rsp.Body.Close()

	if rsp.StatusCode != 200 {
		return "", errors.New(rsp.Status)
	}

	body, err := ioutil.ReadAll(rsp.Body)

	if err != nil {
		return "", err
	}

	var stub interface{}

	err = json.Unmarshal(body, &stub)

	if err != nil {
		return "", err
	}

	body, err = json.Marshal(stub)

	if err != nil {
		return "", err
	}

	hash := md5.Sum(body)
	str_hash := hex.EncodeToString(hash[:])

	return str_hash, nil
}

func CompareHashes(hashes map[string]string) bool {

	last := "-"

	for _, hash := range hashes {

		if last != "-" && last != hash {
			return false
		}

		last = hash
	}

	return true
}

func main() {

	filelist := flag.Bool("filelist", false, "Read WOF IDs from a \"file list\" document.")

	flag.Parse()
	args := flag.Args()

	wofids := make([]*WOFId, 0)

	if *filelist {

		fh, err := os.Open(args[0])

		if err != nil {
			log.Fatal(err)
		}

		defer fh.Close()

		scanner := bufio.NewScanner(fh)

		for scanner.Scan() {

			path := scanner.Text()
			wofid, err := uri.IdFromPath(path)

			if err != nil {
				log.Fatal(err)
			}

			repo, err := uri.RepoFromPath(path)

			if err != nil {
				fmt.Fprintf(os.Stderr, "unable to determine repo from path for WOF ID %d (%s) because %s\n", wofid, path, err)
			}

			w := NewWOFId(wofid, repo)
			wofids = append(wofids, w)
		}

	} else {

		for _, id := range args {

			wofid, err := strconv.ParseInt(id, 10, 64)

			if err != nil {
				log.Fatal(err)
			}

			w := NewWOFId(wofid, "")
			wofids = append(wofids, w)
		}
	}

	if len(wofids) == 0 {
		log.Fatal("Missing WOF ID")
	}

	sources := map[string]string{
		"wof":    "https://whosonfirst.mapzen.com/data/",
		"github": "https://raw.githubusercontent.com/whosonfirst-data/:REPO:/master/data/",
		"s3":     "https://s3.amazonaws.com/whosonfirst.mapzen.com/data/",
	}

	var writer *csv.DictWriter

	for _, w := range wofids {

		hashes, err := HashWOFId(w, sources)

		if err != nil {
			log.Fatal(err)
		}

		match := "MATCH"

		if !CompareHashes(hashes) {
			match = "MISMATCH"
		}

		out := map[string]string{
			"wofid": strconv.FormatInt(w.Id, 10),
			"match": match,
		}

		for src, hash := range hashes {
			out[src] = hash
		}

		if writer == nil {

			fieldnames := make([]string, 0)

			for k, _ := range out {
				fieldnames = append(fieldnames, k)
			}

			writer, err = csv.NewDictWriter(os.Stdout, fieldnames)

			if err != nil {
				log.Fatal(err)
			}

			writer.WriteHeader()
		}

		writer.WriteRow(out)
	}

	os.Exit(0)
}
