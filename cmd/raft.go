package main

import (
	"context"
	"github.com/coconutLatte/go-raft"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"time"
)

func main() {
	rand.Seed(time.Now().Unix())
	go_raft.InitSimpleLog()

	wg := &sync.WaitGroup{}

	ctx, cancel := context.WithCancel(context.Background())
	raftNode, err := go_raft.NewRaftNode(ctx, wg)
	if err != nil {
		go_raft.Errorf("new raft node failed, %v", err)
		os.Exit(1)
	}
	go raftNode.Start()

	wg.Add(1)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt)
		go_raft.Info("listen os interrupt signal")
		<-c
		cancel()
		wg.Done()
	}()
	wg.Wait()
}
