package cs

import "github.com/nyiyui/qrystal/central"

func (s *CentralSource) ReplaceCC(cc *central.Config) {
	s.ccLock.Lock()
	defer s.ccLock.Unlock()
	s.cc = *cc
}

func (s *CentralSource) AddTokens(ts []Token) error {
	for _, t := range ts {
		err := s.Tokens.AddToken(t.Hash, t.Info, true)
		if err != nil {
			return err
		}
	}
	return nil
}
