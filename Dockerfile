FROM golang:1-buster AS build

COPY . /go/src/github.com/nkcmr/inhealth
RUN cd /go/src/github.com/nkcmr/inhealth && \
    go build -v .

FROM debian:buster

COPY --from=build /go/src/github.com/nkcmr/inhealth/inhealth /inhealth

ENTRYPOINT ["/inhealth"]
