package util

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"

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

// MarshalJSON implements json.Marshaler.
func (b *Base64Bytes) MarshalJSON() ([]byte, error) {
	s := base64.StdEncoding.EncodeToString(*b)
	return json.Marshal(s)
}

// UnmarshalJSON implements json.Unmarshaler.
func (b *Base64Bytes) UnmarshalJSON(data []byte) error {
	var raw string
	err := json.Unmarshal(data, &raw)
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

// MarshalJSON implements json.Marshaler.
func (b *HexBytes) MarshalJSON() ([]byte, error) {
	s := hex.EncodeToString(*b)
	return json.Marshal(s)
}

// UnmarshalJSON implements json.Unmarshaler.
func (b *HexBytes) UnmarshalJSON(data []byte) error {
	var raw string
	err := json.Unmarshal(data, &raw)
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
