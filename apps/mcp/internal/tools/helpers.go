package tools

import (
	"net/url"
	"strconv"
)

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

// btoa converts bool to "true" or empty string for false.
func btoa(b bool) string {
	if !b {
		return ""
	}
	return "true"
}
