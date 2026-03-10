package tools

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// rawJSON wraps a JSON string so it marshals as raw JSON (not double-encoded).
type rawJSON string

func (r rawJSON) MarshalJSON() ([]byte, error) {
	if r == "" {
		return []byte("null"), nil
	}
	if !json.Valid([]byte(r)) {
		return nil, fmt.Errorf("rawJSON: value is not valid JSON")
	}
	return []byte(r), nil
}

// queryParams builds a URL query string from key-value pairs, skipping empty values.
func queryParams(pairs ...string) string {
	if len(pairs)%2 != 0 {
		return ""
	}
	v := url.Values{}
	for i := 0; i < len(pairs); i += 2 {
		if pairs[i+1] != "" {
			v.Set(pairs[i], pairs[i+1])
		}
	}
	if len(v) == 0 {
		return ""
	}
	return "?" + v.Encode()
}

// itoa converts int to string, returning empty string for zero.
func itoa(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

// btoa converts bool to string, returning empty string for false.
func btoa(b bool) string {
	if !b {
		return ""
	}
	return "true"
}
