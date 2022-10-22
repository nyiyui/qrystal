package cs

import (
	"crypto/sha256"
	"fmt"
	"sync"
)

type sha256Sum = [sha256.Size]byte

type tokenStore struct {
	tokens     map[[sha256.Size]byte]TokenInfo
	tokensLock sync.RWMutex
}

func newTokenStore() tokenStore {
	return tokenStore{
		tokens: map[[sha256.Size]byte]TokenInfo{},
	}
}

func (s *tokenStore) AddToken(sum sha256Sum, info TokenInfo, overwrite bool) (alreadyExists bool) {
	s.tokensLock.Lock()
	defer s.tokensLock.Unlock()
	_, ok := s.tokens[sum]
	if ok && !overwrite {
		return true
	}
	s.tokens[sum] = info
	return false
}

func (s *tokenStore) getToken(token string) (info TokenInfo, ok bool) {
	sum := sha256.Sum256([]byte(token))
	s.tokensLock.RLock()
	defer s.tokensLock.RUnlock()
	info, ok = s.tokens[sum]
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
