package config

// Subprocess configures a subprocess.
type Subprocess struct {
	Credential Credential `yaml:"credential"`
	Path       string     `yaml:"path"`
}

// Mio configures a Mio process.
type Mio struct {
	Subprocess Subprocess `yaml:"subprocess"`
}

// Node configures a Node process.
type Node struct {
	Subprocess Subprocess `yaml:"subprocess"`
	Dir        string     `yaml:"working-dir"`
	ConfigPath string     `yaml:"config-path"`
}

// Root is configured by the config file.
type Root struct {
	Mio    Mio  `yaml:"mio"`
	Hokuto Mio  `yaml:"hokuto"`
	Node   Node `yaml:"node"`
}
