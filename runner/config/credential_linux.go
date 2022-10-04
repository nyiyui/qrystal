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
	User  string `yaml:"user"`
	Group string `yaml:"group"`
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
	return &syscall.Credential{
		Uid: uint32(uid),
		Gid: uint32(gid),
	}, nil
}
