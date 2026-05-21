

include build.conf

init:
	go mod tidy

dev:
	go run ./src/main.go


# 需要集成发布器 goreleaser
build: 
# Mac
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X seeed-cli/commands/configs.Version=$(VERSION)" -o bin/seeed-cli-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X seeed-cli/commands/configs.Version=$(VERSION)" -o bin/seeed-cli-darwin-arm64
# Linux
	GOOS=linux GOARCH=amd64 go build -ldflags "-X seeed-cli/commands/configs.Version=$(VERSION)" -o bin/seeed-cli-linux-amd64 
	GOOS=linux GOARCH=arm64 go build -ldflags "-X seeed-cli/commands/configs.Version=$(VERSION)" -o bin/seeed-cli-linux-arm64 
# Windows
	GOOS=windows GOARCH=amd64 go build -ldflags "-X seeed-cli/commands/configs.Version=$(VERSION)" -o bin/seeed-cli.exe 


run:
	./bin/seeed-cli

install:
	go install -ldflags "-X seeed-cli/commands/configs.Version=$(VERSION)"

# 	安装后确认是否安装成功
# 	ls $(go env GOPATH)/bin

# 如命令在 go env GOPATH 目录下，则可以直接执行，但是无法执行，请将 go env GOPATH 目录添加到环境变量 PATH 中
# echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.zshrc
# source ~/.zshrc


clean:
	rm -rf ./bin