FROM  golang:1.5.3

# env
ENV ENVIRONMENT ${ENVIRONMENT:-development}
ENV INSTALLDIR  ${INSTALLDIR:-/wof}
ENV DATADIR     ${DATADIR:-/data}
ENV METADIR     ${METADIR:-/meta}
ENV METAFILES   ${METAFILES:-"wof-country-latest wof-locality-latest wof-county-latest wof-localadmin-latest wof-neighborhood-latest wof-region-latest"}
ENV METAURL     ${METAURL:-"https://raw.githubusercontent.com/whosonfirst/whosonfirst-data/master/meta"}
ENV SOURCEURL   ${SOURCEURL:-"http://s3.amazonaws.com/whosonfirst.mapzen.com"}
ENV HOST        ${HOST:-"localhost"}
ENV PORT        ${PORT:-"9999"}

EXPOSE ${PORT}

# setup
RUN mkdir ${INSTALLDIR}
RUN mkdir ${METADIR}
VOLUME ["${DATADIR}"]
WORKDIR ${INSTALLDIR}
COPY ./docker/download_data.sh ${WORKDIR}
COPY ./docker/run.sh ${WORKDIR}
ADD . ${INSTALLDIR}

# build
RUN apt-get update -y
RUN apt-get install build-essential -y

RUN make deps
RUN make bin

RUN ./download_data.sh

CMD ./run.sh
