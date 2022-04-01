package main

import (
	"context"
	"fmt"
	raft "github.com/coconutLatte/go-raft"
	"github.com/coconutLatte/go-raft/log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"
)

func main() {
	args := os.Args
	if len(args) < 5 {
		fmt.Println("not enough address, at least 3")
		os.Exit(1)
	}

	dbNum, err := strconv.Atoi(args[1])
	if err != nil {
		fmt.Printf("convert %s to int failed, %v", args[1], err)
		os.Exit(1)
	}

	run(dbNum, args[1+dbNum], args[2:])
}

func run(dbNum int, address string, addresses []string) {
	rand.Seed(time.Now().Unix())
	log.InitSimpleLog()

	wg := &sync.WaitGroup{}

	ctx, cancel := context.WithCancel(context.Background())
	raftNode, err := raft.NewRaftNode(ctx, wg, address, addresses, dbNum)
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
