package mio

import (
	"fmt"
	"io/fs"
	"os"
)

func (sm *Mio) Forwarding(q ForwardingQ, r *string) error {
	switch q.Type {
	case ForwardingTypeIPv4:
		var b []byte
		if q.Enable {
			b = []byte("1\n")
		} else {
			b = []byte("0\n")
		}
		err := os.WriteFile("/proc/sys/net/ipv4/ip_forward", b, fs.ModePerm)
		if err != nil {
			*r = fmt.Sprintf("WriteFile: %s", err)
			return nil
		}
	default:
		return fmt.Errorf("invalid type %d", q.Type)
	}
	return nil
}
