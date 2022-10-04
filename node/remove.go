package node

import (
	"fmt"

	"github.com/nyiyui/qrystal/mio"
)

func (c *Node) RemoveAllDevices() error {
	c.ccLock.RLock()
	defer c.ccLock.RUnlock()
	for nn := range c.cc.Networks {
		err := c.mio.RemoveDevice(mio.RemoveDeviceQ{
			Name: nn,
		})
		if err != nil {
			return fmt.Errorf("mio RemoveDevice %s: %w", nn, err)
		}
	}
	return nil
}
