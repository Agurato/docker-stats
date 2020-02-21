# Docker stats

Simple server displaying stats of your docker containers

## Run using Docker

Run from docker container (**only for armhf**):

```shell
docker run -d --name docker-stats -v /var/run/docker.sock:/var/run/docker.sock -p 11235:11235 -e BASEURL=localhost:11235 agurato/docker-stats:latest
```

Using `docker-compose`:

```yaml
version: '3'

services:
  docker-stats:
    image: agurato/docker-stats:latest
    ports:
      - 11235:11235
    environment:
      - BASEURL=localhost:11235
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    restart: unless-stopped
```

Go to [http://localhost:11235/](http://localhost:11235/) to display docker stats.

## Build & run

```shell
go get github.com/Agurato/docker-stats
go build github.com/Agurato/docker-stats
cd $GOPATH/src/github.com/Agurato/docker-stats
docker-stats
```
