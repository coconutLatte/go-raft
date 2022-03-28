package server

import (
	"context"
	"github.com/coconutLatte/go-raft/pkg/log"
	"github.com/gin-gonic/gin"
	"net/http"
	"sync"
)

type Server struct {
	base *http.Server
	ctx  context.Context
	wg   *sync.WaitGroup
}

func NewServer(ctx context.Context, wg *sync.WaitGroup) (*Server, error) {
	engine := gin.Default()

	registerEndpoints(engine)

	server := &http.Server{
		Handler: engine,
		Addr:    ":8080",
	}

	return &Server{
		base: server,
		ctx:  ctx,
		wg:   wg,
	}, nil
}

func (s *Server) Start() {
	s.wg.Add(1)
	go func() {
		if err := s.base.ListenAndServe(); err != http.ErrServerClosed {
			log.Warn("start server failed, %v", err)
		}
	}()

	log.Info("starting...")
	<-s.ctx.Done()
	s.stop()
}

func (s *Server) stop() {
	log.Debug("shutting down server...")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.base.Shutdown(ctx); err != nil {
		log.Warnf("shut down server failed, %v", err)
	}
	log.Debug("shut down server success!")
	s.wg.Done()
}

var endpoints map[string]Endpoint

func registerEndpoints(engine *gin.Engine) {
	for _, endpoint := range endpoints {
		for method, handler := range endpoint.handlers {
			engine.Handle(method, endpoint.relativePath, handler)
		}
	}
}
