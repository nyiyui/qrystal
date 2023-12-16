package util

import (
	"sync"

	"go.uber.org/zap"
)

var L *zap.Logger
var S *zap.SugaredLogger
var setupLock sync.Mutex

func SetupLog() {
	setupLock.Lock()
	defer setupLock.Unlock()
	L, _ = zap.NewDevelopment()
	S = L.Sugar()
}
