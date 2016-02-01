FROM  golang:1.5.3

# env
ENV ENVIRONMENT ${ENVIRONMENT:-development}
ENV INSTALLDIR  ${INSTALLDIR:-/wof}
ENV DATADIR     ${DATADIR:-/data}
ENV METADIR     ${METADIR:-/meta}
ENV METAFILES   ${METAFILES:-"wof-country-latest"}
ENV METAURL     ${METAURL:-"https://raw.githubusercontent.com/whosonfirst/whosonfirst-data/master/meta"}
ENV SOURCEURL   ${SOURCEURL:-"http://s3.amazonaws.com/whosonfirst.mapzen.com"}
ENV START_CMD   ${SOURCEURL:-"http://s3.amazonaws.com/whosonfirst.mapzen.com"}

EXPOSE 9999

# setup
RUN mkdir ${INSTALLDIR}
RUN mkdir ${DATADIR}
RUN mkdir ${METADIR}
WORKDIR ${INSTALLDIR}
COPY ./docker/download_data.sh ${WORKDIR}
COPY ./docker/run.sh ${WORKDIR}
ADD . ${INSTALLDIR}

# build
RUN apt-get update -y
RUN apt-get install build-essential -y

RUN make deps
RUN make bin

# data
RUN ./download_data.sh

# chown and start
RUN chown -R 9999:9999 ${DATADIR}
RUN chown -R 9999:9999 ${INSTALLDIR}
USER 9999

CMD ./run.sh
