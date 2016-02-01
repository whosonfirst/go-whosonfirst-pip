#!/usr/bin/env bash

# construct the metafile args
for f in ${METAFILES}; do
  csv="${f}.csv "
done

echo ${csv}
./bin/wof-pip-server \
  -strict \
  -loglevel info \
  -host 0.0.0.0 \
  -port 9999 \
  -cache_all \
  -data ${DATADIR} ${METADIR}/${csv}
