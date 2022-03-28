package node

import (
	"context"
	"github.com/coconutLatte/go-raft/pkg/log"
	"github.com/coconutLatte/go-raft/pkg/server"
	"sync"
)

type Role string

const (
	Leader    Role = "leader"
	Candidate Role = "candidate"
	Follower  Role = "follower"
)

type RaftNode struct {
	role string

	ctx    context.Context
	server *server.Server
}

func NewRaftNode(ctx context.Context, wg *sync.WaitGroup) (*RaftNode, error) {
	log.Info("new raft node")
	raftNode := &RaftNode{
		ctx: ctx,
	}

	srv, err := server.NewServer(ctx, wg)
	if err != nil {
		log.Errorf("new http server failed, %v", err)
		return nil, err
	}
	raftNode.server = srv

	return raftNode, nil
}

func (n *RaftNode) Start() {
	n.server.Start()
}
