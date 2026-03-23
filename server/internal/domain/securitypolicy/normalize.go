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

// NormalizeHosts validates, lowercases, trims, and deduplicates a list of outbound hostnames.
func NormalizeHosts(hosts []string) ([]string, error) {
	return normalizeList(hosts, NormalizeHost)
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

// NormalizeHost validates a single outbound hostname.
func NormalizeHost(host string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(host))
	if normalized == "" {
		return "", fmt.Errorf("invalid host %q: value is empty", host)
	}
	if strings.Contains(normalized, "://") || strings.ContainsAny(normalized, "/?#@") {
		return "", fmt.Errorf("invalid host %q: expected hostname without scheme, path, query, or fragment", host)
	}
	if strings.Contains(normalized, ":") {
		return "", fmt.Errorf("invalid host %q: ports are not allowed", host)
	}
	if net.ParseIP(normalized) != nil {
		return "", fmt.Errorf("invalid host %q: use hostnames instead of IP addresses", host)
	}
	if normalized == "localhost" {
		return normalized, nil
	}

	labels := strings.Split(normalized, ".")
	if len(labels) < 2 {
		return "", fmt.Errorf("invalid host %q: expected a hostname like api.example.com", host)
	}
	for _, label := range labels {
		if label == "" {
			return "", fmt.Errorf("invalid host %q: empty hostname label", host)
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return "", fmt.Errorf("invalid host %q: hostname labels cannot start or end with '-'", host)
		}
		for _, r := range label {
			switch {
			case r >= 'a' && r <= 'z':
			case r >= '0' && r <= '9':
			case r == '-':
			default:
				return "", fmt.Errorf("invalid host %q: hostname contains invalid character %q", host, r)
			}
		}
	}

	return normalized, nil
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
