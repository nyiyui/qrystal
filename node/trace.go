package node

import (
	"encoding/json"
	"os"
	"runtime/trace"
	"sync"

	"github.com/nyiyui/qrystal/util"
)

var traceUntil []string
var traceCheckLock sync.Mutex
var traceFile *os.File

func TraceInit() {
	outputPath := os.Getenv("QRYSTAL_TRACE_OUTPUT_PATH")
	if outputPath == "" {
		return
	}
	{
		traceUntilCNs := os.Getenv("QRYSTAL_TRACE_UNTIL_CNS")
		traceUntil = make([]string, 0)
		err := json.Unmarshal([]byte(traceUntilCNs), &traceUntil)
		if err != nil {
			util.S.Errorf("trace: failed parse QRYSTAL_TRACE_UNTIL_CNS: %s", err)
			return
		}
	}
	var err error
	traceFile, err = os.Create(outputPath)
	if err != nil {
		util.S.Errorf("trace: failed to create file %s: %s", outputPath, err)
		return
	}
	err = trace.Start(traceFile)
	if err != nil {
		util.S.Errorf("trace: failed to start trace: %s", err)
		return
	}
	util.S.Info("trace: ready.")
}

func (n *Node) traceCheck() {
	if traceUntil == nil {
		// trace not enabled
		return
	}
	traceCheckLock.Lock()
	defer traceCheckLock.Unlock()
	matches := 0
	for key := range n.cc.Networks {
		for _, key2 := range traceUntil {
			if key == key2 {
				matches++
			}
		}
	}
	if matches != len(traceUntil) {
		return
	}
	if !trace.IsEnabled() {
		util.S.Info("trace: already stopped.")
		return
	}
	trace.Stop()
	err := traceFile.Close()
	if err != nil {
		util.S.Errorf("trace: failed to close file (while checking): %s", err)
	}
	util.S.Info("trace: stopped due to QRYSTAL_TRACE_UNTIL_CNS being satisfied")
}
