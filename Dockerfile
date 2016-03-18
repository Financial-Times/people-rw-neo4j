FROM alpine:3.3

ADD *.go /people-rw-neo4j/
ADD people/*.go /people-rw-neo4j/people/

RUN apk add --update bash \
  && apk --update add git bzr \
  && apk --update add go \
  && export GOPATH=/gopath \
  && REPO_PATH="github.com/Financial-Times/people-rw-neo4j" \
  && mkdir -p $GOPATH/src/${REPO_PATH} \
  && cp -r people-rw-neo4j/* $GOPATH/src/${REPO_PATH} \
  && cd $GOPATH/src/${REPO_PATH} \
  && go get -t ./... \
  && go build \
  && mv people-rw-neo4j /app \
  && apk del go git bzr \
  && rm -rf $GOPATH /var/cache/apk/*

CMD exec /app --neo-url=$NEO_URL --port=$APP_PORT --batchSize=$BATCH_SIZE --graphiteTCPAddress=$GRAPHITE_ADDRESS --graphitePrefix=$GRAPHITE_PREFIX --logMetrics=false --env=local
