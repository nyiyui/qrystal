//go:build linux

package config

import (
	"fmt"
	"os/user"
	"strconv"
	"syscall"
)

// Credential is syscall.Credential
type Credential struct {
	User   string   `yaml:"user"`
	Group  string   `yaml:"group"`
	Groups []string `yaml:"groups"`
}

func (c Credential) ToCredential() (*syscall.Credential, error) {
	u, err := user.Lookup(c.User)
	if err != nil {
		return nil, err
	}
	uid, err := strconv.ParseInt(u.Uid, 10, 32)
	if err != nil {
		panic(fmt.Sprintf("parse uid: %s", err))
	}
	g, err := user.LookupGroup(c.Group)
	if err != nil {
		return nil, err
	}
	gid, err := strconv.ParseInt(g.Gid, 10, 32)
	if err != nil {
		panic(fmt.Sprintf("parse gid: %s", err))
	}
	groups := make([]uint32, len(c.Groups))
	for i, grp := range c.Groups {
		g, err := user.LookupGroup(grp)
		if err != nil {
			return nil, fmt.Errorf("group %d %s: %w", i, grp, err)
		}
		gid, err := strconv.ParseInt(g.Gid, 10, 32)
		if err != nil {
			panic(fmt.Sprintf("parse gid: %s", err))
		}
		groups[i] = uint32(gid)
	}
	return &syscall.Credential{
		Uid:    uint32(uid),
		Gid:    uint32(gid),
		Groups: groups,
	}, nil
}
