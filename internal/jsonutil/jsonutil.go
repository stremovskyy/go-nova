package jsonutil

import (
	"bytes"
	"encoding/json"
)

// Marshal encodes v into JSON without HTML escaping and without a trailing newline.
//
// This matters for NovaPay RSA signatures: the x-sign is calculated over the exact
// request body bytes.
func Marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	b := buf.Bytes()
	// json.Encoder.Encode always adds a trailing \n.
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	// Return a copy to prevent accidental modifications.
	out := make([]byte, len(b))
	copy(out, b)
	return out, nil
}

// MustMarshal is a convenience helper for tests/examples.
func MustMarshal(v any) []byte {
	b, err := Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
