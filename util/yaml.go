package util

import (
	"encoding/base64"

	"gopkg.in/yaml.v3"
)

// Base64Bytes is a byte slice with methods for (un)marshalling to YAML (using base64.StdEncoding).
type Base64Bytes []byte

// MarshalYAML implements yaml.Marshaler.
func (b Base64Bytes) MarshalYAML() (interface{}, error) {
	return base64.StdEncoding.EncodeToString(b), nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (b Base64Bytes) UnmarshalYAML(value *yaml.Node) error {
	var raw string
	err := value.Decode(&raw)
	if err != nil {
		return err
	}
	bytes, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return err
	}
	copy(b, bytes)
	return nil
}
