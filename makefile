

init:
	go mod tidy

dev:
	go run ./src/main.go


# 需要集成发布器 goreleaser
build: 
# Mac
	GOOS=darwin GOARCH=amd64 go build -o frame-darwin-amd64
# Linux
	GOOS=linux GOARCH=amd64 go build -o frame-linux-amd64 
# Windows
	GOOS=windows GOARCH=amd64 go build -o frame.exe 


run:
	./bin/seeed-cli

install:
	go install

# 	安装后确认是否安装成功
# 	ls $(go env GOPATH)/bin


clean:
	rm -rf ./bin