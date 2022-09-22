package node

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/nyiyui/qanms/node/api"
)

func (n *Node) SpreadCC(ctx context.Context, q *api.SpreadCCQ) (s *api.SpreadCCS, err error) {
	cc, err := newCCFromAPI(q.Signed.Cc)
	if err != nil {
		return nil, err
	}
	ccRaw, err := json.Marshal(cc)
	if err != nil {
		return nil, err
	}
	body := new(bytes.Buffer)
	fmt.Fprintf(body, "%d", q.Signed.Version) // this should be safe as this only has decimals and ccRaw should start with a "{"
	_, _ = body.Write(ccRaw)
	ok := ed25519.Verify(n.centralPubKey, body.Bytes(), q.Signature)
	if !ok {
		return nil, errors.New("signature validation failed")
	}
	n.ccLock.Lock()
	defer n.ccLock.Unlock()
	// TODO: spread to others? perhaps by enumerating CentralPeers.accessible and having a version meter thing on it
	n.cc = *cc
	res, err := n.Sync(context.Background())
	if err != nil {
		return nil, err
	}
	// TODO: check res
	_ = res
	return &api.SpreadCCS{}, nil
}
