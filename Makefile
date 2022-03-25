export GOPROXY=http://goproxy.cn

default:
	go mod tidy
	go build -o bin/raft cmd/raft.go

.PHONY: default