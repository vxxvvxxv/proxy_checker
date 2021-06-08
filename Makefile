.PHONY: clean all

all: clean build

clean:
	@echo "> Clear bin"
	@-rm -rf bin/*

build_darwin_amd64:
	@echo "> Build (darwin - amd64)"
	@go build -race -o ./bin/proxy_checker.darwin-amd64

