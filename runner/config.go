package runner

import (
	_ "embed"
	"os"

	"github.com/nyiyui/qrystal/runner/config"
)

var NodeUser = "qrystal-node"

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
		Hokuto: config.Mio{
			Subprocess: config.Subprocess{
				Credential: config.Credential{
					User:  "root",
					Group: "root",
				},
				Path: os.Getenv("RUNNER_HOKUTO_PATH"),
			},
		},
		Node: config.Node{
			Subprocess: config.Subprocess{
				Credential: config.Credential{
					User:  NodeUser,
					Group: NodeUser,
				},
				Path: os.Getenv("RUNNER_NODE_PATH"),
			},
			ConfigPath: os.Getenv("RUNNER_NODE_CONFIG_PATH"),
		},
	}
}
