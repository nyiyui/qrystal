package node

import (
	"fmt"
	"log"
	"net/rpc"

	"github.com/nyiyui/qrystal/mio"
)

type mioHandle struct {
	client *rpc.Client
	token  []byte
}

func newMio(port uint16, token []byte) (*mioHandle, error) {
	client, err := rpc.DialHTTP("tcp", fmt.Sprintf("127.0.0.1:%d", port))
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
	log.Printf("%#v", q)
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
