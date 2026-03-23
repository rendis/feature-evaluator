package externalapi

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// ValidateWithAllowedHosts runs the standard validation and, when configured,
// enforces that the outbound URL host is static and explicitly allowed.
func ValidateWithAllowedHosts(api *ExternalAPI, allowedHosts []string) error {
	if err := Validate(api); err != nil {
		return err
	}

	return ValidateURLTemplateHost(api.Request.URLTemplate, allowedHosts)
}

// ValidateURLTemplateHost validates the configured URL template host against the allowlist.
func ValidateURLTemplateHost(urlTemplate string, allowedHosts []string) error {
	if len(allowedHosts) == 0 {
		return nil
	}
	for _, match := range collectMatches(urlTemplate) {
		if match.URLKind != nil && *match.URLKind == URLKindDomain {
			return apierror.NewBadRequest(
				"external api host must be static when EXTERNAL_API_ALLOW_HOSTS is configured",
				"error.invalidExternalAPIRequest",
			)
		}
	}

	parsed, err := parseAbsoluteURL(urlTemplate)
	if err != nil {
		return err
	}

	return validateAllowedHost(parsed.Hostname(), allowedHosts)
}

// ValidateRenderedURLHost validates the final rendered outbound URL host against the allowlist.
func ValidateRenderedURLHost(rawURL string, allowedHosts []string) error {
	if len(allowedHosts) == 0 {
		return nil
	}

	parsed, err := parseAbsoluteURL(rawURL)
	if err != nil {
		return err
	}

	return validateAllowedHost(parsed.Hostname(), allowedHosts)
}

func parseAbsoluteURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, apierror.NewBadRequest("external api url template is invalid", "error.invalidExternalAPIRequest")
	}
	if parsed.Scheme == "" || parsed.Host == "" || parsed.Hostname() == "" {
		return nil, apierror.NewBadRequest(
			"external api url template must be an absolute URL",
			"error.invalidExternalAPIRequest",
		)
	}

	return parsed, nil
}

func validateAllowedHost(host string, allowedHosts []string) error {
	normalizedHost := strings.ToLower(strings.TrimSpace(host))
	for _, allowedHost := range allowedHosts {
		if normalizedHost == allowedHost {
			return nil
		}
	}

	return apierror.NewBadRequest(
		fmt.Sprintf("external api host %q is not allowed", normalizedHost),
		"error.externalAPIHostNotAllowed",
	)
}
