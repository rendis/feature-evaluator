package securitypolicy

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// NormalizeOrigins validates, lowercases, trims, and deduplicates a list of browser origins.
func NormalizeOrigins(origins []string) ([]string, error) {
	return normalizeList(origins, NormalizeOrigin)
}

// NormalizeOrigin validates a single origin.
func NormalizeOrigin(origin string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(origin))
	if err != nil {
		return "", fmt.Errorf("invalid origin %q: %w", origin, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid origin %q: expected scheme://host[:port]", origin)
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("invalid origin %q: must not include credentials, query, or fragment", origin)
	}
	if parsed.Path != "" {
		return "", fmt.Errorf("invalid origin %q: must not include a path", origin)
	}
	if parsed.Opaque != "" {
		return "", fmt.Errorf("invalid origin %q: opaque origins are not supported", origin)
	}

	parsed.Scheme = strings.ToLower(parsed.Scheme)
	host := strings.ToLower(parsed.Hostname())
	if port := parsed.Port(); port != "" {
		parsed.Host = net.JoinHostPort(host, port)
	} else {
		parsed.Host = host
	}

	return parsed.String(), nil
}
func normalizeList(values []string, normalize func(string) (string, error)) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}

		nextValue, err := normalize(trimmed)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[nextValue]; exists {
			continue
		}
		seen[nextValue] = struct{}{}
		normalized = append(normalized, nextValue)
	}

	return normalized, nil
}
