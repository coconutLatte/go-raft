package go_raft

import (
	"github.com/coconutLatte/go-raft/log"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

func init() {
	ept := NewEndpoint(
		WithEndpointName("hello"),
		WithEndpointPath("/hello"),
		WithEndpointGet(func(context *gin.Context) {
			context.String(http.StatusOK, "hello\n")
		}))
	err := Register(ept)
	if err != nil {
		panic("register endpoint hello failed")
	}

	ept = NewEndpoint(
		WithEndpointName("vote"),
		WithEndpointPath("/vote"),
		WithEndpointPost(requestVote))
	err = Register(ept)
	if err != nil {
		panic("register endpoint vote failed")
	}

	ept = NewEndpoint(
		WithEndpointName("heartbeat"),
		WithEndpointPath("/heartbeat"),
		WithEndpointGet(heartbeat))
	err = Register(ept)
	if err != nil {
		panic("register endpoint heartbeat failed")
	}
}

func requestVote(ginCtx *gin.Context) {
	log.Info("request vote")

	raftNode := GetRaftNode(ginCtx)

	// only follower and not voted can vote for somebody
	if raftNode.role == Follower && !raftNode.voted.Load() {
		raftNode.voted.Store(true)
		ginCtx.JSON(http.StatusOK, nil)
		return
	}

	// deny request vote
	ginCtx.JSON(http.StatusForbidden, nil)
}

func GetRaftNode(ginCtx *gin.Context) *RaftNode {
	raftNodeVal, exist := ginCtx.Get("raft_node")
	if !exist {
		log.Warn("gin ctx['raft_node'] not exist")
	}

	return raftNodeVal.(*RaftNode)
}

type Heartbeat struct {
	Address string    `json:"address"`
	Time    time.Time `json:"time"`
}

func heartbeat(ginCtx *gin.Context) {
	log.Info("receive heartbeat")

	raftNode := GetRaftNode(ginCtx)

	raftNode.resetCh <- 1

	ginCtx.JSON(http.StatusOK, &Heartbeat{
		Address: raftNode.address,
		Time:    time.Now(),
	})
}
