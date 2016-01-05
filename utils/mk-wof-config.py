#!/usr/bin/env python

import sys
import os
import json
import random
import logging
import socket

import mapzen.whosonfirst.placetypes

if __name__ == '__main__':

    import optparse
    opt_parser = optparse.OptionParser()

    opt_parser.add_option('-w', '--wof', dest='wof', action='store', default=None, help='The path to your Who\'s On First data repository')
    opt_parser.add_option('-o', '--out', dest='out', action='store', default=None, help='Where to write your config file, if "-" then config will be written to STDOUT')
    opt_parser.add_option('-r', '--roles', dest='roles', action='store', default='common,common_optional,optional', help='List of Who\'s On First placetype roles to include (default is "common,common_optional,optional"')
    opt_parser.add_option('-e', '--exclude', dest='exclude', action='store', default='', help='List of Who\'s On First placetypes to exclude, even if they are part of a role (default is None')
    opt_parser.add_option('-v', '--verbose', dest='verbose', action='store_true', default=False, help='Be chatty (default is false)')

    options, args = opt_parser.parse_args()

    if options.verbose:
        logging.basicConfig(level=logging.DEBUG)
    else:
        logging.basicConfig(level=logging.INFO)

    roles = options.roles.split(",")
    exclude = options.exclude.split(",")

    impossible = ('venue', 'address', 'planet', 'building')

    for pt in impossible:
        if not pt in exclude:
            exclude.append(pt)
        
    wof = os.path.abspath(options.wof)
    meta = os.path.join(wof, 'meta')

    config = []
    ports = []

    for pt in mapzen.whosonfirst.placetypes.with_roles(roles) :

        # Is this a valid placetype?

        if pt in exclude:
            logging.debug("%s is in exclude list, skipping" % pt)
            continue

        fname = "wof-%s-latest.csv" % pt
        path = os.path.join(meta, fname)

        if not os.path.exists(path):
            logging.warning("meta file for %s (%s) does not exist, skipping" % (pt, path))
            continue

        # Pick a port! Any port!!

        port = None

        while not port or port in ports:

            port = random.randint(1025, 49151)

            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            result = sock.connect_ex(('127.0.0.1', port))

            if result == 0:
                logging.debug("port %s is already in use, trying again" % port)
                port = None

        # Add it to the list

        config.append({
            'Target': pt,
            'Port': port,
            'Meta': path
        })

    # All done

    if options.out == "-":
        fh = sys.stdout
    else:
        out = os.path.abspath(options.out)
        fh = open(out, 'w')

    json.dump(config, fh, indent=2)
    sys.exit(0)
