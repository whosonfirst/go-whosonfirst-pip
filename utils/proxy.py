#!/usr/bin/env python

import sys
import os
import subprocess
import logging
import json
import signal
import time
import urllib2

def ping(url):

    req = urllib2.Request(url)
    req.get_method = lambda : 'HEAD'

    try:
        urllib2.urlopen(req)
        return True
    except urllib2.HTTPError, e:
        return True
    except Exception, e:
        return False

if __name__ == '__main__':

    import optparse
    opt_parser = optparse.OptionParser()

    opt_parser.add_option('-d', '--data', dest='data', action='store', default=None, help='The path to your Who\'s On First data')
    opt_parser.add_option('--proxy-config', dest='proxy_config', action='store', default=None, help='')
    opt_parser.add_option('--proxy-host', dest='proxy_host', action='store', default='localhost', help='')
    opt_parser.add_option('--proxy-port', dest='proxy_port', action='store', default='1111', help='')

    opt_parser.add_option('--pip-server', dest='pip_server', action='store', default=None, help='')
    opt_parser.add_option('--proxy-server', dest='proxy_server', action='store', default=None, help='')

    opt_parser.add_option('-v', '--verbose', dest='verbose', action='store_true', default=False, help='Be chatty (default is false)')
    options, args = opt_parser.parse_args()

    if options.verbose:
        logging.basicConfig(level=logging.DEBUG)
    else:
        logging.basicConfig(level=logging.INFO)

    whoami = os.path.abspath(sys.argv[0])
    utils = os.path.dirname(whoami)
    root = os.path.dirname(utils)

    bin = os.path.join(root, "bin")

    pip_server = options.pip_server
    proxy_server = options.proxy_server

    if pip_server == None:
        pip_server = os.path.join(bin, "wof-pip-server")

    if proxy_server == None:
        proxy_server = os.path.join(bin, "wof-pip-proxy")

    # Make sure there is something to work with 

    for path in (options.data, options.proxy_config, pip_server, proxy_server):

        if not os.path.exists(path):
            logging.error("%s does exist" % path)
            sys.exit(1)

    # Parse the spec/config

    try:
        fh = open(options.proxy_config, "r")
        spec = json.load(fh)
    except Exception, e:
        logging.error("failed to open %s, because %s" % options.proxy_config, e)
        sys.exit(1)

    procs = []

    # Do some basic sanity checking on the config

    for target in spec:

        for prop in ('Target', 'Port', 'Meta'):

            if not target.get(prop, False):
                logging.error("Invalid spec (missing %s)" % prop)
                sys.exit(1)

    # Start all the PIP servers

    for target in spec:

        cmd = [ pip_server, "-cors", "-port", str(target['Port']), "-data", options.data, target['Meta'] ]
        logging.debug(cmd)

        proc = subprocess.Popen(cmd)
        procs.append(proc)

    # Wait for the PIP servers to finish indexing

    while True:
        
        pending = False

        for target in spec:

            url = "http://localhost:%s" % target['Port']
            logging.debug("ping %s" % url)

            if not ping(url):
                logging.info("ping for %s failed, waiting" % url)
                pending = True

        if not pending:
            break

        logging.info("pause...")
        time.sleep(1)

    # Now start the proxy server

    cmd = [ proxy_server, "-host", options.proxy_host, "-port", options.proxy_port, "-config", options.proxy_config ]
    logging.debug(cmd)

    proc = subprocess.Popen(cmd)
    procs.append(proc)

    # Now just sit around waiting for a SIGINT

    def signal_handler(signal, frame):

        for p in procs:
            p.terminate()

        raise Exception, "all done"

    signal.signal(signal.SIGINT, signal_handler)

    try:
        while True:
            time.sleep(.5)
    except Exception, e:
        pass

    # Done

    logging.info("all done")
    sys.exit()
