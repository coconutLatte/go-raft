package go_raft

import (
	"context"
	"github.com/coconutLatte/go-raft/log"
	"github.com/go-resty/resty/v2"
	"go.uber.org/atomic"
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
	//name    string
	address string

	role    Role
	roleMu  *sync.Mutex
	resetCh chan interface{}

	voted *atomic.Bool

	ctx       context.Context
	server    *Server
	clients   []*resty.Client
	addresses []string

	notify *Notify

	wg *sync.WaitGroup
}

func NewRaftNode(ctx context.Context, wg *sync.WaitGroup, addresses []string) (*RaftNode, error) {
	log.Info("new raft node")
	raftNode := &RaftNode{
		role:      Follower,
		roleMu:    &sync.Mutex{},
		ctx:       ctx,
		wg:        wg,
		clients:   make([]*resty.Client, 0),
		notify:    NewNotify(),
		address:   addresses[0],
		addresses: addresses,
	}
	raftNode.initNotify()

	for _, addr := range raftNode.otherNodes() {
		raftNode.clients = append(raftNode.clients, resty.New().SetBaseURL(addr))
	}

	srv, err := NewServer(ctx, wg, raftNode)
	if err != nil {
		log.Errorf("new http server failed, %v", err)
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
			log.Debug(n.role.toString())
			//log.Debug("new self exam round...")
			// TODO check self status to determine what to do

			tc.Reset(randomTCTime())
		case <-n.resetCh:
			//log.Debug("clear timeout")
			// TODO clear time out

			tc.Reset(randomTCTime())
		case <-n.ctx.Done():
			//log.Debug("break self exam success!")
			n.wg.Done()
			break
		}
	}
}

func (n *RaftNode) doCheck(tc *time.Ticker) {
	n.roleMu.Lock()
	defer n.roleMu.Unlock()
	if n.role == Follower {
		log.Info("promote to candidate")
		n.role = Candidate
		// TODO requestVote to other raft nodes
		n.requestVoteFromOtherNodes()
	}

	if n.role == Leader {
		// TODO do heartbeat
		log.Info("I'm leader, going to do heartbeat round")

		tc.Reset(1 * time.Second)
	}
}

func (n *RaftNode) requestVoteFromOtherNodes() {
	// self voted
	votes := 1
	for _, client := range n.clients {
		resp, err := client.R().Post("/vote")
		if err != nil {
			log.Errorf("post /vote failed, %v", err)
			continue
		}

		log.Infof("resp %#v", resp)
		if resp.IsSuccess() {
			votes++
		}
	}

	if votes > len(n.addresses)/2 {
		n.role = Leader
	}
}

func (n *RaftNode) otherNodes() []string {
	selfIndex := 0
	shouldDel := false
	for i, addr := range n.addresses {
		if addr == n.address {
			selfIndex = i
			shouldDel = true
		}
	}

	if shouldDel {
		return append(n.addresses[:selfIndex], n.addresses[selfIndex+1:]...)
	}

	log.Warnf("not found node address [%s] in addresses %v", n.address, n.addresses)
	return nil
}

const baseTime = 1 * time.Second

// 2.0s ~ 2.1s
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
	log.Debugf("change role from %s to %s", n.role.toString(), m["role"])

	n.roleMu.Lock()
	n.role = Roles()[m["role"]]
	n.roleMu.Unlock()
}
