all:
	test lint

test:
	go test ./... -v

lint:
	$(GOPATH)/bin/golangci-lint run ./... --fast --enable-all

tracker:
	go run ./cmd/tracker

seeder:
	go run ./cmd/seeder

client:
	go run ./cmd/client

clean:
	rm -rf build
build:
	mkdir -p build
	go build -o build/message.linux-amd64 ./message/
	go build -o build/client.linux-amd64 ./cmd/client/
	go build -o build/seeder.linux-amd64 ./cmd/seeder/
	go build -o build/tracker.linux-amd64 ./cmd/tracker/

