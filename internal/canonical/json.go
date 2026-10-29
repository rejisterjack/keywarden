package canonical

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

// Bytes encodes v as compact JSON with object keys sorted.
// The same logical value always produces the same bytes.
func Bytes(v any) ([]byte, error) {
	norm, err := normalize(v)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(norm); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// Hash is SHA-256 over the canonical encoding of v.
func Hash(v any) ([]byte, error) {
	b, err := Bytes(v)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(b)
	return sum[:], nil
}

// FromJSON canonicalizes a JSON document.
func FromJSON(raw []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("trailing JSON")
		}
		return nil, err
	}
	return Bytes(v)
}

func normalize(v any) (any, error) {
	switch t := v.(type) {
	case nil:
		return nil, nil
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			n, err := normalize(val)
			if err != nil {
				return nil, err
			}
			out[k] = n
		}
		return out, nil
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			n, err := normalize(val)
			if err != nil {
				return nil, err
			}
			out[i] = n
		}
		return out, nil
	case json.Number:
		return t, nil
	case string, bool:
		return t, nil
	case int:
		return json.Number(strconv.Itoa(t)), nil
	case int64:
		return json.Number(strconv.FormatInt(t, 10)), nil
	case float64:
		return json.Number(strconv.FormatFloat(t, 'f', -1, 64)), nil
	default:
		raw, err := json.Marshal(t)
		if err != nil {
			return nil, fmt.Errorf("cannot canonicalize %T", v)
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		var out any
		if err := dec.Decode(&out); err != nil {
			return nil, err
		}
		return normalize(out)
	}
}
