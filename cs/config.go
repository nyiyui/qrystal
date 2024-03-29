package cs

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/util"
	"gopkg.in/yaml.v3"
)

type TLS struct {
	CertPath string `yaml:"certPath"`
	KeyPath  string `yaml:"keyPath"`
}

type Config struct {
	Addr         string          `yaml:"addr"`
	RyoAddr      string          `yaml:"ryoAddr"`
	TLS          TLS             `yaml:"tls"`
	CC           *central.Config `yaml:"central"`
	Tokens       *TokensConfig   `yaml:"tokens"`
	BackportPath string          `yaml:"backportPath"`
	DBPath       string          `yaml:"dbPath"`
}

func (c *Config) apply() {
	if c.BackportPath == "" {
		c.BackportPath = filepath.Join(os.Getenv("STATE_DIRECTORY"), "cs-backport.yml")
	}
	if c.DBPath == "" {
		c.DBPath = filepath.Join(os.Getenv("STATE_DIRECTORY"), "db")
	}
}

type TokensConfig struct {
	Raw []Token
}

func (t *TokensConfig) UnmarshalJSON(data []byte) error {
	var raw []TokenConfig
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}
	t2, err := convertTokens2(raw)
	if err != nil {
		return err
	}
	t.Raw = t2
	return nil
}

func (t *TokensConfig) UnmarshalYAML(value *yaml.Node) error {
	var raw []TokenConfig
	err := value.Decode(&raw)
	if err != nil {
		return err
	}
	t2, err := convertTokens2(raw)
	if err != nil {
		return err
	}
	t.Raw = t2
	return nil
}

type TokenConfig struct {
	Name             string                 `yaml:"name" json:"name"`
	Hash             *util.TokenHash        `yaml:"hash" json:"hash"`
	Networks         map[string]string      `yaml:"networks" json:"networks"`
	CanPull          bool                   `yaml:"canPull"`
	CanPush          *CanPush               `yaml:"canPush"`
	CanAdminTokens   *CanAdminTokens        `yaml:"canAdminTokens"`
	CanSRVUpdate     bool                   `yaml:"canSRVUpdate"`
	SRVAllowances    []central.SRVAllowance `yaml:"srvAllowances"`
	SRVAllowancesAny bool                   `yaml:"srvAllowancesAny"`
}

func convertTokens2(tokens []TokenConfig) ([]Token, error) {
	res := make([]Token, len(tokens))
	for i, token := range tokens {
		res[i] = Token{
			Hash: *token.Hash,
			Info: TokenInfo{
				Name:             token.Name,
				Networks:         token.Networks,
				CanPull:          token.CanPull,
				CanPush:          token.CanPush,
				CanAdminTokens:   token.CanAdminTokens,
				CanSRVUpdate:     token.CanSRVUpdate,
				SRVAllowances:    token.SRVAllowances,
				SRVAllowancesAny: token.SRVAllowancesAny,
			},
		}
	}
	return res, nil
}

func LoadConfig(configPath string) (*Config, error) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("config read: %s", err)
	}
	var config Config
	err = yaml.Unmarshal(raw, &config)
	if err != nil {
		return nil, fmt.Errorf("config unmarshal: %s", err)
	}
	if config.CC == nil {
		return nil, errors.New("no central config specified")
	}
	if len(config.CC.Networks) == 0 {
		return nil, errors.New("no central networks specified")
	}
	for cnn, cn := range config.CC.Networks {
		if cn.Me != "" {
			return nil, fmt.Errorf("net %s: me is not blank", cnn)
		}
	}
	config.apply()
	return &config, nil
}

type Backport struct {
	CC     *central.Config   `yaml:"cc"`
	Tokens map[string]string `yaml:"tokens"`
}

func (s *CentralSource) backportSilent() {
	go s.backportSilentInternal()
}

func (s *CentralSource) backportSilentInternal() {
	// TODO: perhaps prevent many backports from piling up
	defer func() {
		r := recover()
		if r != nil {
			util.S.Errorf("backportSilent: recover: %s", r)
		}
	}()
	if s == nil {
		util.S.Errorf("backportSilent: CentralSource is nil!")
		return
	}
	err := s.backport()
	if err != nil {
		util.S.Errorf("backportSilent: error: %s", err)
	}
}

func (s *CentralSource) backport() error {
	s.backportLock.Lock()
	defer s.backportLock.Unlock()
	var encoded []byte
	tokens, err := s.Tokens.convertToMap()
	if err != nil {
		return nil
	}
	func() {
		s.ccLock.RLock()
		defer s.ccLock.RUnlock()
		encoded, err = yaml.Marshal(Backport{
			CC:     &s.cc,
			Tokens: tokens,
		})
	}()
	if err != nil {
		return err
	}
	err = os.WriteFile(s.backportPath, encoded, 0o0600)
	if err != nil {
		return err
	}
	return nil
}

func (s *CentralSource) ReadBackport() error {
	s.backportLock.Lock()
	defer s.backportLock.Unlock()
	encoded, err := os.ReadFile(s.backportPath)
	if err != nil {
		return err
	}
	var b Backport
	err = yaml.Unmarshal(encoded, &b)
	if err != nil {
		return err
	}
	if b.CC != nil {
		func() {
			s.ccLock.Lock()
			defer s.ccLock.Unlock()
			s.cc = *b.CC
		}()
	}
	return nil
}
