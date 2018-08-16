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
	go build example/main.go

clean:
	rm -f main
