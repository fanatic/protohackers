# syntax=docker/dockerfile:1

FROM golang:1.19-alpine

WORKDIR /app

RUN apk add --no-cache supervisor
RUN apk add --update coreutils && rm -rf /var/cache/apk/*
COPY supervisord.conf /app/supervisord.conf
CMD ["/usr/bin/supervisord", "-c", "/app/supervisord.conf"]

COPY . .

RUN go mod download
RUN mkdir bin
RUN go build  -o bin ./...
