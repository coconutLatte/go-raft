package main

import (
	"github.com/coconutLatte/go-raft/pkg/log"
	"github.com/coconutLatte/go-raft/pkg/node"
)

func main() {
	log.InitSimpleLog()
	raftNode := node.NewRaftNode()
	raftNode.Start()
}
