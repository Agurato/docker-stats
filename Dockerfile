FROM arm32v7/golang:1.13.8

RUN go get github.com/docker/docker/api/types
RUN go get github.com/docker/docker/client
RUN go get github.com/elastic/go-sysinfo
RUN go get github.com/google/uuid
RUN go get github.com/gorilla/websocket

COPY . /go/src/github.com/Agurato/docker-stats

RUN go install github.com/Agurato/docker-stats

WORKDIR /go/src/github.com/Agurato/docker-stats

CMD [ "/go/bin/docker-stats" ]