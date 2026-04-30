dev:
	go run ./src/main.go

build:
	go build -o ./bin/seeed-cli ./src/main.go

run:
	./bin/seeed-cli

install:
	go mod tidy

clean:
	rm -rf ./bin