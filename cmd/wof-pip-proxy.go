package main

import (
	"bufio"
	_ "bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-log"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	_ "strconv"
	"strings"
)

type WOFProxyTargets map[string]WOFProxyTarget

type WOFProxyTarget struct {
	Target string
	Port   int
	Host   string
	Meta   string
}

func (pt WOFProxyTarget) URL() string {

	scheme := "http"
	host := "localhost"

	if pt.Host != "" {
		host = pt.Host
	}

	port := pt.Port

	return fmt.Sprintf("%s://%s:%d", scheme, host, port)
}

func (pt WOFProxyTarget) Ping() (bool, error) {

	/*
	   Note that wof-pip-server does not have a true 'ping' endpoint
	   so we are just checking that it returns anything at all
	   (20160104/thisisaaronland)
	*/

	test := pt.URL()
	req, err := http.NewRequest("HEAD", test, nil)

	if err != nil {
		return false, err
	}

	client := &http.Client{}
	_, err = client.Do(req)

	if err != nil {
		return false, err
	}

	return true, nil
}

type WOFProxyHandler func(rsp http.ResponseWriter, req *http.Request)

func main() {

	var host = flag.String("host", "localhost", "The hostname to listen for requests on")
	var port = flag.Int("port", 1111, "The port number to listen for requests on")
	var config = flag.String("config", "", "... (If the value is - then read the config from STDIN)")
	var loglevel = flag.String("loglevel", "info", "Log level for reporting")
	var logs = flag.String("logs", "", "Where to write logs to disk")

	flag.Parse()

	var l_writer io.Writer
	l_writer = io.MultiWriter(os.Stdout)

	if *logs != "" {

		l_file, l_err := os.OpenFile(*logs, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)

		if l_err != nil {
			panic(l_err)
		}

		l_writer = io.MultiWriter(os.Stdout, l_file)
	}

	logger := log.NewWOFLogger("[wof-pip-server] ")
	logger.AddLogger(l_writer, *loglevel)

	var spec []byte

	if *config == "-" {

		var raw string

		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			raw += scanner.Text()
		}

		spec = []byte(raw)

	} else {

		_spec, err := ioutil.ReadFile(*config)

		if err != nil {
			logger.Error("Failed to read %s, because %v", *config, err)
			os.Exit(1)
		}

		// Oh Go...
		spec = _spec
	}

	targets, err := proxyHandlerTargets(spec)

	if err != nil {
		logger.Error("Failed to parse config, because %v", err)
		os.Exit(1)
	}

	for t, p := range targets {

		if p.Port == *port {
			logger.Error("Target port (%s:%d) is the same as proxy port", t, *port)
			os.Exit(1)
		}

		/*

			You might be thinking: Oh wouldn't it be nice to spawn a series of subprocesses
			but it appears as though Go deliberately chooses _not_ to support subprocesses
			which is kind of weird but there you go... Maybe it's possible and I just don't
			know the magic incantation but for now we're going to assume that starting the
			individual servers is something else's problem... See also `utils/proxy.py` for
			precisely this reason... Good times... (20160104/thisisaaronland)

		*/

		ok, err := p.Ping()

		if !ok {
			logger.Error("Target (%s:%d) is not awake or connected to the network: %v", p.Host, p.Port, err)
			os.Exit(1)
		}

		fmt.Println(p.Target, p.URL())
	}

	handler := proxyHandlerFunc(targets)
	proxyHandler := http.HandlerFunc(handler)

	endpoint := fmt.Sprintf("%s:%d", *host, *port)
	logger.Info("proxying requests at %s\n", endpoint)

	http.ListenAndServe(endpoint, proxyHandler)
	os.Exit(0)
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

		// please just make me a url.URL thingy...

		url := target.URL() + "?" + req.URL.RawQuery

		/*
			src := req.URL.Path
			dst := url

			fmt.Printf("proxy %s to %s\n", src, dst)
		*/

		_req, err := http.NewRequest("GET", url, nil)

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
