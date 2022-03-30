package go_raft

import (
	"context"
	"github.com/go-resty/resty/v2"
	"math/rand"
	"sync"
	"time"
)

type Role int

const (
	Follower Role = iota
	Candidate
	Leader
)

func Roles() map[string]Role {
	return map[string]Role{
		"follower":  Follower,
		"candidate": Candidate,
		"leader":    Leader,
	}
}

func (r Role) toString() string {
	return map[Role]string{
		Follower:  "follower",
		Candidate: "candidate",
		Leader:    "leader",
	}[r]
}

type RaftNode struct {
	role    Role
	resetCh chan interface{}

	ctx    context.Context
	server *Server
	client *resty.Client
	notify *Notify

	wg *sync.WaitGroup
}

func NewRaftNode(ctx context.Context, wg *sync.WaitGroup) (*RaftNode, error) {
	Info("new raft node")
	raftNode := &RaftNode{
		role:   Follower,
		ctx:    ctx,
		wg:     wg,
		client: resty.New(),
		notify: NewNotify(),
	}
	raftNode.initNotify()

	srv, err := NewServer(ctx, wg)
	if err != nil {
		Errorf("new http server failed, %v", err)
		return nil, err
	}
	raftNode.server = srv

	return raftNode, nil
}

func (n *RaftNode) Start() {
	go n.server.Start()

	go n.selfExam()
}

// run a loop clock to decide what to do itself
func (n *RaftNode) selfExam() {
	n.wg.Add(1)
	tc := time.NewTicker(randomTCTime())
	for {
		select {
		case <-tc.C:
			Debug("new self exam round...")
			// TODO check self status to

			tc.Reset(randomTCTime())
		case <-n.resetCh:
			Debug("clear timeout")
			// TODO clear time out

			tc.Reset(randomTCTime())
		case <-n.ctx.Done():
			Debug("break self exam success!")
			n.wg.Done()
			break
		}
	}
}

const baseTime = 1 * time.Second

// 1.0s ~ 1.1s
func randomTCTime() time.Duration {
	return baseTime + time.Duration(rand.Intn(100))*time.Millisecond
}

type Notify struct {
	listeners map[string]func(map[string]string)
}

func NewNotify() *Notify {
	n := &Notify{
		listeners: make(map[string]func(map[string]string)),
	}

	return n
}

func (n *Notify) Call(event string, m map[string]string) {
	n.listeners[event](m)
}

func (n *RaftNode) initNotify() {
	n.notify.listeners["changeRole"] = n.changeRole
}

func (n *RaftNode) changeRole(m map[string]string) {
	Debugf("change role from %s to %s", n.role.toString(), m["role"])
	n.role = Roles()[m["role"]]
}
