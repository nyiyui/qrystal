package cs

import (
	"crypto/sha256"
	"sync"
)

type tokenStore struct {
	tokens     map[[sha256.Size]byte]TokenInfo
	tokensLock sync.RWMutex
}

func (s *tokenStore) getToken(token string) (info TokenInfo, ok bool) {
	sum := sha256.Sum256([]byte(token))
	s.tokensLock.RLock()
	defer s.tokensLock.RUnlock()
	info, ok = s.tokens[sum]
	return
}

type TokenInfo struct {
	Name    string
	CanPull bool
	CanPush bool
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
