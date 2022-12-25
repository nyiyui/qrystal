package runner

import (
	_ "embed"
	"os"

	"github.com/nyiyui/qrystal/runner/config"
)

const QrystalNodeUsername = "qrystal-node"

var nodeUser string

func newConfig() config.Root {
	return config.Root{
		Mio: config.Mio{
			Subprocess: config.Subprocess{
				Credential: config.Credential{
					User:  "root",
					Group: "root",
				},
				Path: os.Getenv("RUNNER_MIO_PATH"),
			},
		},
		Node: config.Node{
			Subprocess: config.Subprocess{
				Credential: config.Credential{
					User:  nodeUser,
					Group: nodeUser,
				},
				Path: os.Getenv("RUNNER_NODE_PATH"),
			},
			ConfigPath: os.Getenv("RUNNER_NODE_CONFIG_PATH"),
		},
	}
}
