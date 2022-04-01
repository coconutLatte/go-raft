package go_raft

import (
	"context"
	"encoding/json"
	"fmt"
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
	db        *DB

	notify *Notify

	wg *sync.WaitGroup

	cst *time.Location
}

func NewRaftNode(ctx context.Context, wg *sync.WaitGroup, address string, addresses []string, dbNum int) (*RaftNode, error) {
	log.Info("new raft node")
	raftNode := &RaftNode{
		roleMu:    &sync.Mutex{},
		resetCh:   make(chan interface{}),
		ctx:       ctx,
		wg:        wg,
		clients:   make([]*resty.Client, 0),
		notify:    NewNotify(),
		address:   address,
		addresses: addresses,
		voted:     &atomic.Bool{},
		cst:       time.FixedZone("CST", 8*3600),
	}
	raftNode.initNotify()

	db, err := OpenDB(dbNum)
	if err != nil {
		log.Errorf("open db failed, %v", err)
		return nil, err
	}
	raftNode.db = db

	nodeInfos := db.GetNodeInfos()
	if len(nodeInfos) == 0 {
		// should init table
		for _, address := range addresses {
			nodeInfos = append(nodeInfos, NodeInfo{Address: address, Role: Follower})
		}

		if err = db.CreateNodeInfos(nodeInfos); err != nil {
			log.Errorf("create node infos %v failed, %v", nodeInfos, err)
			return nil, err
		}

		raftNode.role = Follower
	} else {
		for _, nodeInfo := range nodeInfos {
			if nodeInfo.Address == raftNode.address {
				raftNode.role = nodeInfo.Role
			}
		}
	}
	log.Debugf("node %s init as role %s", raftNode.address, raftNode.role.toString())

	for _, addr := range raftNode.otherNodes() {
		raftNode.clients = append(raftNode.clients, resty.New().SetBaseURL("http://"+addr))
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
Loop:
	for {
		select {
		case <-tc.C:
			n.doCheck(tc)
		case <-n.resetCh:
			log.Debug("clear timeout")
			log.Debugf("[%s]", n.role.toString())
			tc.Reset(randomTCTime())
		case <-n.ctx.Done():
			break Loop
		}
	}
	n.wg.Done()
}

func (n *RaftNode) doCheck(tc *time.Ticker) {
	n.roleMu.Lock()
	defer n.roleMu.Unlock()
	if n.role == Follower {
		log.Info("promote to candidate")
		n.role = Candidate
		n.requestVoteFromOtherNodes()
	}

	if n.role == Leader {
		log.Info("I'm leader, going to heartbeat round")
		n.heartbeatRound()
		tc.Reset(1 * time.Second)
	} else {
		tc.Reset(randomTCTime())
	}
}

func (n *RaftNode) requestVoteFromOtherNodes() {
	// self voted
	votes := 1
	n.voted.Store(true)
	for _, client := range n.clients {
		resp, err := client.R().Post("/vote")
		if err != nil {
			log.Errorf("post /vote failed, %v", err)
			continue
		}

		if resp.IsSuccess() {
			votes++
		}
	}

	if votes > len(n.addresses)/2 {
		n.updateRole(Leader)
		leaderNode, _ := n.db.GetLeaderNode()
		if leaderNode == nil {
			if err := n.db.CreateLeaderNode(&LeaderNode{
				Address:     n.address,
				LeaderRound: 1,
			}); err != nil {
				log.Errorf("create leader node failed, %v", err)
			}
		} else {
			if err := n.db.UpdateLeaderNode(&LeaderNode{
				Address:     n.address,
				LeaderRound: leaderNode.LeaderRound + 1,
			}); err != nil {
				log.Errorf("increase all leader round failed, %v", err)
			}
		}
	} else {
		n.updateRole(Follower)
		n.voted.Store(false)
	}
}

func (n *RaftNode) heartbeatRound() {
	leaderNodeInfo, err := n.db.GetLeaderNode()
	if err != nil {
		log.Errorf("get leader node %s info failed, %v", n.address, err)
		return
	}

	if leaderNodeInfo.Address != n.address {
		log.Error("leaderNodeInfo.Address != n.address")
		return
	}

	var hbs []HeartbeatRsp
	for _, client := range n.clients {
		resp, err := client.R().
			SetBody(&HeartbeatReq{
				LeaderAddress: leaderNodeInfo.Address,
				LeaderRound:   leaderNodeInfo.LeaderRound,
			}).
			Post("/heartbeat")
		if err != nil {
			log.Errorf("get /heartbeat failed, %v", err)
			continue
		}

		hb := &HeartbeatRsp{}
		if err = json.Unmarshal(resp.Body(), hb); err != nil {
			log.Errorf("convert resp body to HeartbeatRsp failed, %v", err)
			continue
		}
		hbs = append(hbs, *hb)
	}

	log.Info(n.formatHBs(hbs))
	log.Info("heartbeat round complete")
}

func (n *RaftNode) formatHBs(hbs []HeartbeatRsp) string {

	hbsStr := ""
	for i, hb := range hbs {
		hbsStr += "Address: " + hb.Address + ", "
		hbsStr += "Time: " + hb.Time.In(n.cst).Format(time.RFC3339)
		if hb.ErrMsg != "" {
			hbsStr += ", ErrMsg: " + hb.ErrMsg
		}
		if i != len(hbs)-1 {
			hbsStr += "; "
		}
	}

	return fmt.Sprintf("{%s}", hbsStr)
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

const baseTime = 5 * time.Second

// 5.0s ~ 5.1s
func randomTCTime() time.Duration {
	return baseTime + time.Duration(rand.Intn(100))*time.Millisecond
}

type Notify struct {
	listeners map[string]func(map[string]string)
}

func (n *RaftNode) updateRole(role Role) {
	n.role = role
	if err := n.db.UpdateNodeRole(n.address, role); err != nil {
		log.Errorf("update node %s to role %s failed, %v", n.address, role.toString(), err)
	}
}

// NewNotify future to decouple need
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
	n.updateRole(Roles()[m["role"]])
	n.roleMu.Unlock()
}
