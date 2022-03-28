package main

import (
	"context"
	"github.com/coconutLatte/go-raft/pkg/log"
	"github.com/coconutLatte/go-raft/pkg/node"
	"os"
	"os/signal"
	"sync"
)

func main() {
	log.InitSimpleLog()
	wg := &sync.WaitGroup{}

	ctx, cancel := context.WithCancel(context.Background())
	raftNode, err := node.NewRaftNode(ctx, wg)
	if err != nil {
		log.Errorf("new raft node failed, %v", err)
		os.Exit(1)
	}
	go raftNode.Start()

	wg.Add(1)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt)
		log.Info("listen os interrupt signal")
		<-c
		cancel()
		wg.Done()
	}()
	wg.Wait()
}
