FROM golang:1.9.0 as builder
WORKDIR /go/src/github.com/moooofly/opencensus-go-exporter-hunter
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o cc_server example/callchain_example/callchain_server/main.go

# Final image.
FROM scratch
LABEL maintainer "moooofly <centos.sf@gmail.com>"
COPY --from=builder /go/src/github.com/moooofly/opencensus-go-exporter-hunter/cc_server .

COPY config.fake .

ENTRYPOINT ["/cc_server"]
CMD ["-h"]
