package logger

import (
	"go.uber.org/zap"
)

var Log *zap.Logger

func InitLogger(level string) {
	var cfg zap.Config
	if level == "debug" {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}
	
	var err error
	Log, err = cfg.Build()
	if err != nil {
		panic(err)
	}
}