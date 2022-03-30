package main

import (
	"context"
	"fmt"
	raft "github.com/coconutLatte/go-raft"
	"github.com/coconutLatte/go-raft/log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"time"
)

func main() {
	args := os.Args
	if len(args) < 4 {
		fmt.Println("not enough address, at least 3")
		os.Exit(1)
	}

	run(args[1:])
}

func run(addresses []string) {
	rand.Seed(time.Now().Unix())
	log.InitSimpleLog()

	wg := &sync.WaitGroup{}

	ctx, cancel := context.WithCancel(context.Background())
	raftNode, err := raft.NewRaftNode(ctx, wg, addresses)
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
