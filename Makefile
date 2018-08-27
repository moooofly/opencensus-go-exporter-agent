all:
	@echo "Usage:"
	@echo "  1. make tcp"
	@echo "  2. make unix"
	@echo "  3. make both"

both: build
	./main -tcp_addr tcp://0.0.0.0:12345 -unix_sock_addr unix:///var/run/hunter-agent.sock

tcp: build
	./main -tcp_addr tcp://0.0.0.0:12345

unix: build
	./main -unix_sock_addr unix:///var/run/hunter-agent.sock

build:
	CGO_ENABLED=0 GOOS=linux go build -o main example/local_example/main.go

build_grpc:
	CGO_ENABLED=0 GOOS=linux go build -o grpc_client example/grpc_example/helloworld_client/main.go
	CGO_ENABLED=0 GOOS=linux go build -o grpc_server example/grpc_example/helloworld_server/main.go

docker: build_grpc
	docker build -t grpc_helloworld_server:v1 -f Dockerfile.hws .
	docker build -t grpc_helloworld_client:v1 -f Dockerfile.hwc .

docker_run:
	@# docker0 (bridge) ->  172.17.0.1
	docker run -d --rm -p 50051:50051 grpc_helloworld_server:v1 -agent_tcp_addr 172.17.0.1:12345 -grpc_server_listen_port 50051
	docker run -d --rm grpc_helloworld_client:v1 -agent_tcp_addr 172.17.0.1:12345 -grpc_server_listen_addr 172.17.0.1:50051

clean:
	rm -f main
	rm -f grpc_client
	rm -f grpc_server
