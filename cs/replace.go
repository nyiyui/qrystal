package cs

import "github.com/nyiyui/qrystal/node"

func (s *CentralSource) ReplaceCC(cc *node.CentralConfig) {
	s.ccLock.Lock()
	defer s.ccLock.Unlock()
	s.cc = *cc
}

func (s *CentralSource) AddTokens(ts []Token) error {
	for _, t := range ts {
		_, err := s.tokens.AddToken(t.Hash, t.Info, true)
		if err != nil {
			return err
		}
	}
	return nil
}
