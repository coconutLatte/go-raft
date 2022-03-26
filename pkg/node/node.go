package node

import (
	"context"
	"github.com/coconutLatte/go-raft/pkg/log"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Role string

const (
	Leader    Role = "leader"
	Candidate Role = "candidate"
	Follower  Role = "follower"
)

type RaftNode struct {
	role string

	server *http.Server

	ctx context.Context
}

func NewRaftNode() *RaftNode {
	engine := gin.Default()

	ctx, cancel := context.WithCancel(context.Background())

	engine.GET("/ping", func(ctx *gin.Context) {
		ctx.String(200, "pong")
		cancel()
	})

	server := &http.Server{
		Handler: engine,
		Addr:    ":8080",
	}

	return &RaftNode{
		server: server,
		ctx:    ctx,
	}
}

func (n *RaftNode) Start() {
	go func() {
		if err := n.server.ListenAndServe(); err != nil {
			log.Warnf("start server failed, %v", err)
		}
	}()

	<-n.ctx.Done()
	n.stop()
}

func (n *RaftNode) stop() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := n.server.Shutdown(ctx); err != nil {
		log.Warnf("shut down server failed, %v", err)
	}
}
