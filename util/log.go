package util

import (
	"log"

	"go.uber.org/zap"
)

var L *zap.Logger
var S *zap.SugaredLogger

func SetupLog() {
	L, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("zap: %s", err)
	}
	defer L.Sync()
	S = L.Sugar()
}
