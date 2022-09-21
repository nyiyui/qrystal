//go:build linux

package config

// Credential is syscall.Credential
type Credential struct {
	UID         uint32   `yaml:"uid"`
	GID         uint32   `yaml:"gid"`
	Groups      []uint32 `yaml:"groups"`
	NoSetGroups bool     `yaml:"no-set-groups"`
}
