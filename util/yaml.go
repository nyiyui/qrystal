package util

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"errors"

	"gopkg.in/yaml.v3"
)

// Base64Bytes is a byte slice with methods for (un)marshalling to YAML (using base64.StdEncoding).
type Base64Bytes []byte

// MarshalYAML implements yaml.Marshaler.
func (b *Base64Bytes) MarshalYAML() (interface{}, error) {
	return base64.StdEncoding.EncodeToString(*b), nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (b *Base64Bytes) UnmarshalYAML(value *yaml.Node) error {
	var raw string
	err := value.Decode(&raw)
	if err != nil {
		return err
	}
	bytes, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return err
	}
	*b = bytes
	return nil
}

// HexBytes is a byte slice with methods for (un)marshalling to YAML (using hex).
type HexBytes []byte

// MarshalYAML implements yaml.Marshaler.
func (b *HexBytes) MarshalYAML() (interface{}, error) {
	return hex.EncodeToString(*b), nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (b *HexBytes) UnmarshalYAML(value *yaml.Node) error {
	var raw string
	err := value.Decode(&raw)
	if err != nil {
		return err
	}
	bytes, err := hex.DecodeString(raw)
	if err != nil {
		return err
	}
	*b = bytes
	return nil
}

// Ed25519PublicKey is a byte slice with methods for (un)marshalling to YAML (using base64.StdEncoding).
type Ed25519PublicKey ed25519.PublicKey

// MarshalYAML implements yaml.Marshaler.
func (b *Ed25519PublicKey) MarshalYAML() (interface{}, error) {
	return base64.StdEncoding.EncodeToString(*b), nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (b *Ed25519PublicKey) UnmarshalYAML(value *yaml.Node) error {
	var raw string
	err := value.Decode(&raw)
	if err != nil {
		return err
	}
	if raw[0] != 'U' || raw[1] != '_' {
		return errors.New("must start with U_")
	}
	raw = raw[2:]
	bytes, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return err
	}
	if len(bytes) != ed25519.PublicKeySize {
		return errors.New("invalid size")
	}
	*b = bytes
	return nil
}
