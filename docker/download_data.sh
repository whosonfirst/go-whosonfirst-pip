#!/usr/bin/env bash

for f in ${METAFILES}; do
  echo "Now working on ${f}:"

  cd ${METADIR}
  echo -e "\t...pulling metadata..."
  wget --quiet -O ${f}.csv ${SOURCEURL}/bundles/${f}.csv

  cd ${DATADIR}
  echo -e "\t...pulling data..."
  wget --quiet -O ${f}.tar.bz2 ${SOURCEURL}/bundles/${f}-bundle.tar.bz2

  echo -e "\t...extracting data..."
  bunzip2 ${f}.tar.bz2 && tar xf ${f}.tar --strip-components=2 && rm ${f}.tar
done
