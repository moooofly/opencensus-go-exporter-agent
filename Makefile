VERSION := v0.7.0

all:
	@echo "Usage:"
	@echo "  1. make tcp"
	@echo "  2. make unix"
	@echo "  3. make both"

both: build
	./main -tcp_addr tcp://0.0.0.0:12345 -unix_sock_addr unix:///var/run/hunter-agent.sock >/dev/null 2>&1

tcp: build
	./main -tcp_addr tcp://0.0.0.0:12345 >/dev/null 2>&1

unix: build
	./main -unix_sock_addr unix:///var/run/hunter-agent.sock >/dev/null 2>&1

build: build_local build_grpc build_cc

build_local:
	CGO_ENABLED=0 GOOS=linux go build -o main example/local_example/main.go

build_grpc:
	CGO_ENABLED=0 GOOS=linux go build -o grpc_client example/grpc_example/helloworld_client/main.go
	CGO_ENABLED=0 GOOS=linux go build -o grpc_server example/grpc_example/helloworld_server/main.go

build_cc:
	CGO_ENABLED=0 GOOS=linux go build -o cc_client example/callchain_example/callchain_client/main.go
	CGO_ENABLED=0 GOOS=linux go build -o cc_server example/callchain_example/callchain_server/main.go

docker: build_grpc build_cc
	@# helloworld grpc
	docker build -t hunter-demo-golang-server:${VERSION} -f example/grpc_example/helloworld_server/Dockerfile .
	docker build -t hunter-demo-golang-client:${VERSION} -f example/grpc_example/helloworld_client/Dockerfile .
	@# challchain grpc
	docker build -t hunter-demo-golang-cc-server:${VERSION} -f example/callchain_example/callchain_server/Dockerfile .
	docker build -t hunter-demo-golang-cc-client:${VERSION} -f example/callchain_example/callchain_client/Dockerfile .

docker_push: docker
	@# helloworld grpc
	docker tag hunter-demo-golang-server:${VERSION} stag-reg.llsops.com/backend/hunter-demo-golang-server:${VERSION}
	docker tag hunter-demo-golang-client:${VERSION} stag-reg.llsops.com/backend/hunter-demo-golang-client:${VERSION}
	docker push stag-reg.llsops.com/backend/hunter-demo-golang-server:${VERSION}
	docker push stag-reg.llsops.com/backend/hunter-demo-golang-client:${VERSION}
	@# challchain grpc
	docker tag hunter-demo-golang-cc-server:${VERSION} stag-reg.llsops.com/backend/hunter-demo-golang-cc-server:${VERSION}
	docker tag hunter-demo-golang-cc-client:${VERSION} stag-reg.llsops.com/backend/hunter-demo-golang-cc-client:${VERSION}
	docker push stag-reg.llsops.com/backend/hunter-demo-golang-cc-server:${VERSION}
	docker push stag-reg.llsops.com/backend/hunter-demo-golang-cc-client:${VERSION}

docker_run: docker
	@# docker0 (bridge) ->  172.17.0.1
	docker run -d --rm --name grpc_server -p 50051:50051 hunter-demo-golang-server:${VERSION} -agent_tcp_ip 172.17.0.1 -grpc_server_listen_port 50051 -configPath config.fake
	docker run -d --rm --name grpc_client hunter-demo-golang-client:${VERSION} -agent_tcp_ip 172.17.0.1 -grpc_server_listen_addr 172.17.0.1:50051 -configPath config.fake

docker_stop:
	docker stop grpc_client
	docker stop grpc_server

tmp: build_grpc
	docker build -t hunter-demo-golang-server:tmp -f example/grpc_example/helloworld_server/Dockerfile .
	docker build -t hunter-demo-golang-client:tmp -f example/grpc_example/helloworld_client/Dockerfile .
	@# docker0 (bridge) ->  172.17.0.1
	docker run -d --rm --name grpc_server -p 50051:50051 hunter-demo-golang-server:tmp -agent_tcp_ip 172.17.0.1 -grpc_server_listen_port 50051 -configPath config.fake
	docker run -d --rm --name grpc_client hunter-demo-golang-client:tmp -agent_tcp_ip 172.17.0.1 -grpc_server_listen_addr 172.17.0.1:50051 -configPath config.fake

clean:
	rm -f main
	rm -f grpc_client
	rm -f grpc_server
	rm -f cc_client
	rm -f cc_server
