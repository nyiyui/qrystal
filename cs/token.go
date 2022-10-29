package cs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tidwall/buntdb"
)

const tokenPrefix = "token:"

type sha256Sum = [sha256.Size]byte

type TokenStore struct {
	db *buntdb.DB
}

func newTokenStore(db *buntdb.DB) (TokenStore, error) {
	err := db.CreateIndex("tokens", "token:*", buntdb.IndexString)
	return TokenStore{
		db: db,
	}, err
}

var errCannotOverwrite = errors.New("cannot overwrite")

func (s *TokenStore) AddToken(sum sha256Sum, info TokenInfo, overwrite bool) (err error) {
	encoded, err := json.Marshal(info)
	if err != nil {
		return
	}
	key := tokenPrefix + hex.EncodeToString(sum[:])
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
	return
}

func (s *TokenStore) GetTokenByHash(hashHex string) (info TokenInfo, ok bool, err error) {
	key := tokenPrefix + hashHex
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
	ok = true
	return
}
func (s *TokenStore) getToken(token string) (info TokenInfo, ok bool, err error) {
	sum := sha256.Sum256([]byte(token))
	key := tokenPrefix + hex.EncodeToString(sum[:])
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
	Name         string
	Networks     map[string]string
	CanPull      bool
	CanPush      *CanPush
	CanAddTokens *CanAddTokens
}

type CanAddTokens struct {
	CanPull bool `yaml:"can-pull"`
	CanPush bool `yaml:"can-push"`
	// don't allow CanAddTokens to make logic simpler
}

type CanPush struct {
	Any      bool              `yaml:"any"`
	Networks map[string]string `yaml:"networks"`
}

type Token struct {
	Hash [sha256.Size]byte
	Info TokenInfo
}

func convertTokens(tokens []Token) map[[sha256.Size]byte]TokenInfo {
	m := map[[sha256.Size]byte]TokenInfo{}
	for _, token := range tokens {
		m[token.Hash] = token.Info
	}
	return m
}

func newTokenAuthError(token string) error {
	sum := sha256.Sum256([]byte(token))
	return fmt.Errorf("token auth failed with hash %x", sum)
}
