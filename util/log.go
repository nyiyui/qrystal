package util

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var L *zap.Logger
var S *zap.SugaredLogger
var Atom zap.AtomicLevel
var setupLock sync.Mutex

func SetupLog() {
	setupLock.Lock()
	defer setupLock.Unlock()
	cfg := zap.NewDevelopmentConfig()
	Atom = zap.NewAtomicLevel()
	L := zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(cfg.EncoderConfig),
		zapcore.Lock(os.Stdout),
		Atom,
	))
	S = L.Sugar()
}
