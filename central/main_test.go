package central

import (
	"bytes"
	"encoding/gob"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGobEncoding(t *testing.T) {
	cc := Config{
		Networks: map[string]*Network{
			"testnet": &Network{
				Name: "sasara",
			},
		},
	}
	buf := new(bytes.Buffer)
	err := gob.NewEncoder(buf).Encode(cc)
	if err != nil {
		t.Fatalf("Encode: %s", err)
	}
	var cc2 Config
	err = gob.NewDecoder(buf).Decode(&cc2)
	if err != nil {
		t.Fatalf("Decode: %s", err)
	}
	if !cmp.Equal(cc, cc2) {
		t.Log(cmp.Diff(cc, cc2))
		t.Fatal("!equal")
	}
}
