package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type WOFProxyTargets map[string]WOFProxyTarget

type WOFProxyTarget struct {
	Target string
	Port   int
}

type WOFProxyHandler func(rsp http.ResponseWriter, req *http.Request)

func main() {

	var host = flag.String("host", "localhost", "The hostname to listen for requests on")
	var port = flag.Int("port", 1111, "The port number to listen for requests on")
	var config = flag.String("config", "", "")

	flag.Parse()

	endpoint := fmt.Sprintf("%s:%d", *host, *port)

	// Check if config == "-" and read from flag.Args()

	spec, err := ioutil.ReadFile(*config)

	if err != nil {
	   panic(err)
	}

	targets, err := proxyHandlerTargets(spec)

	if err != nil {
		panic(err)
	}

	for t, p := range targets {
		if p.Port == *port {
			err := fmt.Sprintf("Target port (%s:%d) is the same as proxy port", t, *port)
			panic(err)
		}
	}

	handler := proxyHandlerFunc(targets)

	proxyHandler := http.HandlerFunc(handler)
	http.ListenAndServe(endpoint, proxyHandler)
}

func proxyHandlerTargets(spec []byte) (WOFProxyTargets, error) {

	var pt []WOFProxyTarget

	err := json.Unmarshal(spec, &pt)

	if err != nil {
		return nil, err
	}

	targets := WOFProxyTargets{}

	for _, p := range pt {
		targets[p.Target] = p
	}

	return targets, nil
}

func proxyHandlerFunc(targets WOFProxyTargets) WOFProxyHandler {

	// See this - it's basically just so that we can scope
	// targets to the handler function (20160104/thisisaaronland)

	return func(rsp http.ResponseWriter, req *http.Request) {

		p := req.URL.Path
		p = strings.Replace(p, "/", "", 1)

		if p == "" {
			http.Error(rsp, "Missing target", http.StatusBadRequest)
			return
		}

		target, ok := targets[p]

		if !ok {
			http.Error(rsp, "Invalid target", http.StatusBadRequest)
			return
		}

		host := fmt.Sprintf("http://localhost:%s", strconv.Itoa(target.Port))
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
}
