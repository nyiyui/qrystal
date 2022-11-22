package node

import (
	"fmt"
	"net/rpc"
	"strings"

	"github.com/nyiyui/qrystal/mio"
)

type mioHandle struct {
	client *rpc.Client
	token  []byte
}

func newMio(addr string, token []byte) (*mioHandle, error) {
	tokens := strings.SplitN(addr, ":", 2)
	client, err := rpc.DialHTTP(tokens[0], tokens[1])
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}
	return &mioHandle{
		client: client,
		token:  token,
	}, nil
}

// ConfigureDevice requests mio to configure the WireGuard device.
//
// Note: q.Name will have "qrystal-" prepended for the WireGuard name.
func (h *mioHandle) ConfigureDevice(q mio.ConfigureDeviceQ) (err error) {
	var errString string
	q.Token = h.token
	err = h.client.Call("Mio.ConfigureDevice", q, &errString)
	if err != nil {
		return fmt.Errorf("call: %w", err)
	}
	if errString != "" {
		return fmt.Errorf("content: %s", errString)
	}
	return nil
}

func (h *mioHandle) RemoveDevice(q mio.RemoveDeviceQ) (err error) {
	var errString string
	q.Token = h.token
	err = h.client.Call("Mio.RemoveDevice", q, &errString)
	if err != nil {
		return fmt.Errorf("call: %w", err)
	}
	if errString != "" {
		return fmt.Errorf("content: %s", errString)
	}
	return nil
}

func (h *mioHandle) Forwarding(q mio.ForwardingQ) (err error) {
	var errString string
	q.Token = h.token
	err = h.client.Call("Mio.Forwarding", q, &errString)
	if err != nil {
		return fmt.Errorf("call: %w", err)
	}
	if errString != "" {
		return fmt.Errorf("content: %s", errString)
	}
	return nil
}
