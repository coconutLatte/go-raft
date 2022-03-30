package go_raft

import (
	"fmt"
	"go.uber.org/zap"
)

var log *zap.SugaredLogger

func InitSimpleLog() {
	logger, err := zap.NewDevelopment(zap.AddCallerSkip(1))
	if err != nil {
		panic(fmt.Sprintf("init zap logger failed, %v", err))
	}

	log = logger.Sugar()
}

func Info(args ...interface{}) {
	log.Info(args)
}

func Infof(template string, args ...interface{}) {
	log.Infof(template, args)
}

func Debug(args ...interface{}) {
	log.Debug(args)
}

func Debugf(template string, args ...interface{}) {
	log.Debugf(template, args)
}

func Warn(args ...interface{}) {
	log.Warn(args)
}

func Warnf(template string, args ...interface{}) {
	log.Warnf(template, args)
}

func Error(args ...interface{}) {
	log.Error(args)
}

func Errorf(template string, args ...interface{}) {
	log.Errorf(template, args)
}
