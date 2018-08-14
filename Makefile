all: run

run: build
	./main -tcp_addr tcp://0.0.0.0:12345 -unix_sock_addr unix:///var/run/hunter-agent.sock

build:
	go build example/main.go

clean:
	rm -f main
