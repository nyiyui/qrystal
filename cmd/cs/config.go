package main

import (
	"github.com/nyiyui/qrystal/cs"
	"github.com/nyiyui/qrystal/node"
	"github.com/nyiyui/qrystal/util"
	"gopkg.in/yaml.v3"
)

type Config struct {
	TLSCertPath string              `yaml:"tls-cert-path"`
	TLSKeyPath  string              `yaml:"tls-key-path"`
	CC          *node.CentralConfig `yaml:"central"`
	Tokens      *Tokens             `yaml:"tokens"`
}

type Tokens struct {
	raw []cs.Token
}

func (t *Tokens) UnmarshalYAML(value *yaml.Node) error {
	var raw []Token
	err := value.Decode(&raw)
	if err != nil {
		return err
	}
	t2, err := convertTokens(raw)
	if err != nil {
		return err
	}
	t.raw = t2
	return nil
}

type Token struct {
	Name         string            `yaml:"name"`
	Hash         *util.HexBytes    `yaml:"hash"`
	Networks     map[string]string `yaml:"networks"`
	CanPull      bool              `yaml:"can-pull"`
	CanPush      bool              `yaml:"can-push"`
	CanAddTokens *cs.CanAddTokens  `yaml:"can-add-tokens"`
}
