FROM golang:alpine

RUN apk update && apk add gcc g++ make git
COPY go.mod go.sum /deps/
RUN cd /deps && go mod download
COPY . /gopatch
WORKDIR /gopatch 
ENTRYPOINT make ci
