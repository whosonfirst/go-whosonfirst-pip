package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

func main() {

	var host = flag.String("host", "localhost", "The hostname to listen for requests on")
	var port = flag.Int("port", 1111, "The port number to listen for requests on")

	flag.Parse()

	endpoint := fmt.Sprintf("%s:%d", *host, *port)

	proxyHandler := http.HandlerFunc(proxyHandlerFunc)
	http.ListenAndServe(endpoint, proxyHandler)
}

func proxyHandlerFunc(rsp http.ResponseWriter, req *http.Request) {

	targets := make(map[string]int)
	targets["locality"] = 6666

	p := req.URL.Path
	p = strings.Replace(p, "/", "", 1)

	if p == "" {
		http.Error(rsp, "Missing target", http.StatusBadRequest)
		return
	}

	port, ok := targets[p]

	if !ok {
		http.Error(rsp, "Invalid target", http.StatusBadRequest)
		return
	}

	host := fmt.Sprintf("http://localhost:%s", strconv.Itoa(port))
	url := host + "?" + req.URL.RawQuery

	body := bytes.NewBuffer([]byte(""))

	_req, err := http.NewRequest("GET", url, body)

	if err != nil {
		http.Error(rsp, err.Error(), http.StatusBadRequest)
		return
	}

	client := &http.Client{}
	_rsp, err := client.Do(_req)

	if err != nil {
		http.Error(rsp, err.Error(), http.StatusBadRequest)
		return
	}

	for k, v := range _rsp.Header {
		for _, vv := range v {
			rsp.Header().Add(k, vv)
		}
	}

	rsp.WriteHeader(_rsp.StatusCode)
	result, err := ioutil.ReadAll(_rsp.Body)

	if err != nil {
		http.Error(rsp, err.Error(), http.StatusBadRequest)
		return
	}

	rsp.Write(result)
}
