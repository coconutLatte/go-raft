package server

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type Option interface {
	apply(e *Endpoint)
}

type optionFunc func(e *Endpoint)

func (f optionFunc) apply(e *Endpoint) {
	f(e)
}

func WithEndpointName(name string) Option {
	return optionFunc(func(e *Endpoint) {
		e.name = name
	})
}

func WithEndpointPath(path string) Option {
	return optionFunc(func(e *Endpoint) {
		e.relativePath = path
	})
}

func WithEndpointGet(handler gin.HandlerFunc) Option {
	return optionFunc(func(e *Endpoint) {
		e.handlers[http.MethodGet] = handler
	})
}

func WithEndpointPost(handler gin.HandlerFunc) Option {
	return optionFunc(func(e *Endpoint) {
		e.handlers[http.MethodPost] = handler
	})
}

func WithEndpointDelete(handler gin.HandlerFunc) Option {
	return optionFunc(func(e *Endpoint) {
		e.handlers[http.MethodDelete] = handler
	})
}

func WithEndpointPut(handler gin.HandlerFunc) Option {
	return optionFunc(func(e *Endpoint) {
		e.handlers[http.MethodPut] = handler
	})
}

func WithEndpointPatch(handler gin.HandlerFunc) Option {
	return optionFunc(func(e *Endpoint) {
		e.handlers[http.MethodPatch] = handler
	})
}

type Endpoint struct {
	// should be unique
	name         string
	relativePath string
	handlers     map[string]gin.HandlerFunc
}
