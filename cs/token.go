package cs

import (
	"crypto/sha256"
	"sync"
)

type tokenStore struct {
	tokens     map[[sha256.Size]byte]TokenInfo
	tokensLock sync.RWMutex
}

func (s *tokenStore) getToken(token []byte) (info TokenInfo, ok bool) {
	sum := sha256.Sum256(token)
	s.tokensLock.Lock()
	defer s.tokensLock.Unlock()
	info, ok = s.tokens[sum]
	return
}

type TokenInfo struct {
	Name    string
	CanPull bool
	CanPush bool
}
