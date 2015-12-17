FROM alpine

ADD *.go /kafka-bridge/
ADD start.sh /

RUN apk add --update bash \
  && apk --update add git bzr \
  && echo "http://dl-4.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories \
  && apk --update add go \
  && export GOPATH=/gopath \
  && REPO_PATH="github.com/Financial-Times/coco-kafka-bridge" \
  && mkdir -p $GOPATH/src/${REPO_PATH} \
  && mv /kafka-bridge/* $GOPATH/src/${REPO_PATH} \
  && cd $GOPATH/src/${REPO_PATH} \
  && go get \
  && go test \
  && go build \
  && mv coco-kafka-bridge /coco-kafka-bridge \
  && apk del go git bzr \
  && rm -rf $GOPATH /var/cache/apk/* 

ENTRYPOINT [ "/bin/sh", "-c" ]
CMD [ "/start.sh" ]