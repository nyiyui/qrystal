package cs

import "github.com/nyiyui/qanms/node"

func (s *CentralSource) ReplaceCC(cc *node.CentralConfig) {
	s.ccLock.Lock()
	defer s.ccLock.Unlock()
	s.cc = *cc
}

func (s *CentralSource) ReplaceTokens(tokens []Token) {
	s.tokens.tokensLock.Lock()
	defer s.tokens.tokensLock.Unlock()
	s.tokens.tokens = convertTokens(tokens)
}
