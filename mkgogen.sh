#!/usr/bin/env bash

# To generate:
#
# cd $(go env GOPATH)/src/moooofly/opencensus-go-exporter-hunter
# ./mkgogen.sh

OUTDIR="$(go env GOPATH)/src"

protoc -I=. --go_out=plugins=grpc:$OUTDIR opencensus/proto/exporter/exporter.proto
