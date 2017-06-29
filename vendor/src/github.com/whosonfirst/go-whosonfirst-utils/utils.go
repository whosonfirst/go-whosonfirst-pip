package utils

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"github.com/tidwall/gjson"
	"io/ioutil"
	_ "log"
	"path"
	"strconv"
	"strings"
)

func HashFile(path string) (string, error) {

	body, err := ioutil.ReadFile(path)

	if err != nil {
		return "", err
	}

	hash := HashBytes(body)
	return hash, nil
}

func HashGeomFromFile(path string) (string, error) {

	body, err := ioutil.ReadFile(path)

	if err != nil {
		return "", err
	}

	return HashGeomFromFeature(body)
}

func HashGeomFromFeature(feature []byte) (string, error) {

	geom := gjson.GetBytes(feature, "geometry")
	body, err := json.Marshal(geom.Value())

	if err != nil {
		return "", err
	}

	hash := HashBytes(body)
	return hash, nil
}

func HashFromJSON(raw []byte) (string, error) {

	var geom interface{}

	err := json.Unmarshal(raw, &geom)

	if err != nil {
		return "", err
	}

	body, err := json.Marshal(geom)

	if err != nil {
		return "", err
	}

	hash := HashBytes(body)
	return hash, nil
}

func HashBytes(body []byte) string {

	hash := md5.Sum(body)
	return hex.EncodeToString(hash[:])
}

// THESE ARE ALL DEPRECATED AND REPLACED BY go-whosonfirst-uri

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
