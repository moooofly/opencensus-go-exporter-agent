# Example gRPC server and client with OpenCensus

This example uses:

* gRPC to create an RPC server and client.
* The **modified** OpenCensus gRPC plugin (ocgrpc) to instrument the RPC server and client.
* Use `opencensus-go-exporter-agent` exporter to output stats and traces to Hunter agent.

## Usage

- download

```
$ go get github.com/moooofly/opencensus-go-exporter-agent
```

- build

```
make clean && make build_grpc
```

- run the server

```
./grpc_server
```

- run the client

```
./grpc_client
```

