package go_raft

import "github.com/gin-gonic/gin"

func init() {
	ept := NewEndpoint(
		WithEndpointName("vote"),
		WithEndpointPath("/vote"),
		WithEndpointPost(requestVote))
	err := Register(ept)
	if err != nil {
		panic("register endpoint vote failed")
	}
}

func requestVote(ginCtx *gin.Context) {

}
