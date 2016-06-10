package utils

import (
	"crypto/md5"
	"encoding/hex"
	"io/ioutil"
	"path"
	"strconv"
	"strings"
)

// this should be a woo-woo interface for Local and Remote datastores
// but not today (20151013/thisisaaronland)

func Id2Fname(id int) (fname string) {

	str_id := strconv.Itoa(id)
	fname = str_id + ".geojson"

	return fname
}

func Id2Path(id int) (path string) {

	parts := []string{}
	input := strconv.Itoa(id)

	for len(input) > 3 {

		chunk := input[0:3]
		input = input[3:]
		parts = append(parts, chunk)
	}

	if len(input) > 0 {
		parts = append(parts, input)
	}

	path = strings.Join(parts, "/")
	return path
}

func Id2RelPath(id int) (rel_path string) {

	fname := Id2Fname(id)
	root := Id2Path(id)

	rel_path = root + "/" + fname
	return rel_path
}

func Id2AbsPath(root string, id int) (abs_path string) {

	rel := Id2RelPath(id)
	abs_path = path.Join(root, rel)

	return abs_path
}

func HashFile(path string) (string, error) {

	body, err := ioutil.ReadFile(path)

	if err != nil {
		return "", err
	}

	hash := md5.Sum(body)
	str_hash := hex.EncodeToString(hash[:])

	return str_hash, nil
}
