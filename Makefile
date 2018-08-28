include VERSION.docker

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
	docker build -t hunter-demo-golang-server:${VERSION} -f Dockerfile.hws .
	docker build -t hunter-demo-golang-client:${VERSION} -f Dockerfile.hwc .

docker_run:
	@# docker0 (bridge) ->  172.17.0.1
	docker run -d --rm --name grpc_server -p 50051:50051 hunter-demo-golang-server:${VERSION} -agent_tcp_ip 172.17.0.1 -grpc_server_listen_port 50051 -configPath config.fake
	docker run -d --rm --name grpc_client hunter-demo-golang-client:${VERSION} -agent_tcp_ip 172.17.0.1 -grpc_server_listen_addr 172.17.0.1:50051 -configPath config.fake

docker_push:
	docker tag hunter-demo-golang-server:${VERSION} stag-reg.llsops.com/backend/hunter-demo-golang-server:${VERSION}
	docker tag hunter-demo-golang-client:${VERSION} stag-reg.llsops.com/backend/hunter-demo-golang-client:${VERSION}
	docker push stag-reg.llsops.com/backend/hunter-demo-golang-server:${VERSION}
	docker push stag-reg.llsops.com/backend/hunter-demo-golang-client:${VERSION}

docker_stop:
	docker stop grpc_client
	docker stop grpc_server

clean:
	rm -f main
	rm -f grpc_client
	rm -f grpc_server
