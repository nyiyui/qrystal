package cs

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nyiyui/qrystal/util"
	"github.com/tidwall/buntdb"
	"gopkg.in/yaml.v3"
)

const tokenPrefix = "token:"

type TokenStore struct {
	db *buntdb.DB
	cs *CentralSource
}

func newTokenStore(db *buntdb.DB, cs *CentralSource) (TokenStore, error) {
	err := db.CreateIndex("tokens", "token:*", buntdb.IndexString)
	return TokenStore{
		db: db,
		cs: cs,
	}, err
}

var errCannotOverwrite = errors.New("cannot overwrite")

func (s *TokenStore) UpdateToken(info TokenInfo) (err error) {
	encoded, err := json.Marshal(info)
	if err != nil {
		return
	}
	err = s.db.Update(func(tx *buntdb.Tx) (err error) {
		_, _, err = tx.Set(info.key, string(encoded), nil)
		return
	})
	s.cs.backportSilent()
	return
}

func (s *TokenStore) AddToken(hash util.TokenHash, info TokenInfo, overwrite bool) (err error) {
	encoded, err := json.Marshal(info)
	if err != nil {
		return
	}
	key := tokenPrefix + hash.String()
	err = s.db.Update(func(tx *buntdb.Tx) (err error) {
		_, err = tx.Get(key)
		switch err {
		case buntdb.ErrNotFound:
			goto Write
		case nil:
			if overwrite {
				goto Write
			} else {
				return errCannotOverwrite
			}
		default:
			return
		}
	Write:
		_, _, err = tx.Set(key, string(encoded), nil)
		return
	})
	s.cs.backportSilent()
	return
}

func (s *TokenStore) RemoveToken(hash util.TokenHash) (err error) {
	key := tokenPrefix + hash.String()
	err = s.db.Update(func(tx *buntdb.Tx) (err error) {
		_, err = tx.Delete(key)
		return err
	})
	s.cs.backportSilent()
	return
}

func (s *TokenStore) GetTokenByHash(hashString string) (info TokenInfo, ok bool, err error) {
	key := tokenPrefix + hashString
	var encoded string
	err = s.db.View(func(tx *buntdb.Tx) error {
		encoded, err = tx.Get(key)
		return err
	})
	if err == buntdb.ErrNotFound {
		ok = false
		err = nil
		return
	}
	err = json.Unmarshal([]byte(encoded), &info)
	info.key = key
	ok = true
	return
}
func (s *TokenStore) getToken(token *util.Token) (info TokenInfo, ok bool, err error) {
	key := tokenPrefix + token.Hash().String()
	var encoded string
	err = s.db.View(func(tx *buntdb.Tx) error {
		encoded, err = tx.Get(key)
		return err
	})
	if err == buntdb.ErrNotFound {
		ok = false
		err = nil
		return
	}
	err = json.Unmarshal([]byte(encoded), &info)
	info.key = key
	ok = true
	return
}

func (s *TokenStore) convertToMap() (m map[string]string, err error) {
	m = map[string]string{}
	err = s.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("tokens", func(key, val string) bool {
			m[key] = val
			return true
		})
	})
	return
}

type TokenInfo struct {
	key            string `json:"-"`
	Name           string
	Networks       map[string]string
	CanPull        bool
	CanPush        *CanPush
	CanAdminTokens *CanAdminTokens
	// CanAdminTokens specifies whether this token can add *or remove* tokens.

	Using    bool
	LastUsed time.Time
}

func (ti *TokenInfo) StartUse() {
	ti.Using = true
}

func (ti *TokenInfo) StopUse() {
	ti.Using = true
	ti.LastUsed = time.Now()
}

func (ti *TokenInfo) Use() {
	ti.LastUsed = time.Now()
}

type CanAdminTokens struct {
	CanPull bool `yaml:"canPull"`
	CanPush bool `yaml:"canPush"`
	// don't allow CanAdminTokens to make logic simpler
}

type CanPush struct {
	Any      bool                      `yaml:"any"`
	Networks map[string]CanPushNetwork `yaml:"networks"`
}

type CanPushNetwork struct {
	Name             string
	CanSeeElement    []string
	CanSeeElementAny bool
}

type canPushNetwork struct {
	Name          string      `yaml:"name"`
	CanSeeElement interface{} `yaml:"canSeeElement"`
}

func (c *CanPushNetwork) UnmarshalYAML(value *yaml.Node) error {
	var cpn canPushNetwork
	err := value.Decode(&cpn)
	if err != nil {
		return err
	}
	c.Name = cpn.Name
	switch cse := cpn.CanSeeElement.(type) {
	case string:
		c.CanSeeElementAny = cse == "any"
	case []string:
		c.CanSeeElement = cse
	default:
		return errors.New("canSeeElement must be \"any\" or []string")
	}
	return nil
}

func (c *CanPushNetwork) MarshalYaml() (interface{}, error) {
	var cse interface{}
	if c.CanSeeElementAny {
		cse = "any"
	} else {
		cse = c.CanSeeElement
	}
	if c.CanSeeElementAny && len(c.CanSeeElement) != 0 {
		return nil, errors.New("CanSeeElementAny == true but CanSeeElement is not zero-length")
	}
	return canPushNetwork{
		Name:          c.Name,
		CanSeeElement: cse,
	}, nil
}

type Token struct {
	Hash util.TokenHash
	Info TokenInfo
}

func newTokenAuthError(token util.Token) error {
	return fmt.Errorf("token auth failed with hash %s", token.Hash())
}
