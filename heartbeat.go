package go_raft

import (
	"fmt"
	"github.com/coconutLatte/go-raft/log"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

func init() {
	ept := NewEndpoint(
		WithEndpointName("heartbeat"),
		WithEndpointPath("/heartbeat"),
		WithEndpointPost(heartbeat))
	err := Register(ept)
	if err != nil {
		panic("register endpoint heartbeat failed")
	}
}

type HeartbeatReq struct {
	LeaderAddress string `json:"leader_address"`
	LeaderRound   int    `json:"leader_round"`
}

type HeartbeatRsp struct {
	Address string    `json:"address"`
	Time    time.Time `json:"time"`

	// only has content when heartbeat failed
	ErrMsg string `json:"err_msg"`
}

func heartbeat(ginCtx *gin.Context) {
	log.Info("receive heartbeat")

	raftNode := GetRaftNode(ginCtx)
	raftNode.resetCh <- 1

	heartbeatReq := &HeartbeatReq{}
	heartbeatRsp := &HeartbeatRsp{
		Address: raftNode.address,
		Time:    time.Now(),
	}
	errMsg := ""
	if err := ginCtx.ShouldBind(heartbeatReq); err != nil {
		errMsg = fmt.Sprintf("convert req body to HeartbeatReq failed, %v", err)
		log.Error(errMsg)
		heartbeatRsp.ErrMsg = errMsg
		ginCtx.JSON(http.StatusInternalServerError, heartbeatRsp)
		return
	}

	leaderNode, err := raftNode.db.GetLeaderNode()
	if err != nil {
		if err = raftNode.db.CreateLeaderNode(&LeaderNode{
			Address:     heartbeatReq.LeaderAddress,
			LeaderRound: heartbeatReq.LeaderRound,
		}); err != nil {
			log.Warnf("create leader node failed, %v", err)
		}

		ginCtx.JSON(http.StatusOK, heartbeatRsp)
		return
	}

	if leaderNode.LeaderRound < heartbeatReq.LeaderRound {
		raftNode.db.UpdateNodeRole(leaderNode.Address, Follower)
		raftNode.db.UpdateNodeRole(heartbeatReq.LeaderAddress, Leader)

		if err = raftNode.db.UpdateLeaderNode(&LeaderNode{
			Address:     heartbeatReq.LeaderAddress,
			LeaderRound: heartbeatReq.LeaderRound,
		}); err != nil {
			log.Warnf("update leader node from %s to %s failed, %v",
				leaderNode.Address, heartbeatReq.LeaderAddress, err)
		}

		raftNode.roleMu.Lock()
		if raftNode.role == Leader {
			raftNode.role = Follower
		}
		raftNode.roleMu.Unlock()

		ginCtx.JSON(http.StatusOK, heartbeatRsp)
	} else if leaderNode.LeaderRound == heartbeatReq.LeaderRound {
		ginCtx.JSON(http.StatusOK, heartbeatRsp)
	} else {
		errMsg = fmt.Sprintf("receive an old leader heartbeat [address: %s, leaderRound: %d], current leader is [address: %s, leaderRound: %d]",
			heartbeatReq.LeaderAddress, heartbeatReq.LeaderRound, leaderNode.Address, leaderNode.LeaderRound)
		log.Error(errMsg)
		heartbeatRsp.ErrMsg = errMsg
		ginCtx.JSON(http.StatusInternalServerError, heartbeatRsp)
	}

}
