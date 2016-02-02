#!/usr/bin/env bash

for f in ${METAFILES}; do
  # construct the metafile args
  csv="${METADIR}/${f}.csv ${csv}"

  # extract the data
  echo "Now working on ${f}:"
  echo -e "\t...extracting data..."

  cd ${DATADIR}
  bunzip2 -f ${f}.tar.bz2 && tar xf ${f}.tar --strip-components=2 && rm ${f}.tar
done

cd ${INSTALLDIR}
./bin/wof-pip-server \
  -strict \
  -loglevel info \
  -host ${HOST} \
  -port ${PORT} \
  -cache_all \
  -data ${DATADIR} ${csv}
