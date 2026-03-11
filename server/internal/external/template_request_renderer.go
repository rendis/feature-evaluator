package external

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/rendis/feature-evaluator/internal/domain/externalapi"
)

var (
	fullPlaceholderPattern     = regexp.MustCompile(`^\s*\{\{\s*([^}]+?)\s*\}\}\s*$`)
	templatePlaceholderPattern = regexp.MustCompile(`\{\{\s*([^}]+?)\s*\}\}`)
)

// RenderedRequest contains the fully rendered request that will be executed.
type RenderedRequest struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    any
}

type placeholderValue struct {
	Value    any
	Present  bool
	Required bool
}

// RenderExternalAPIRequest builds the reusable external API request using typed params and secrets.
func RenderExternalAPIRequest(
	request externalapi.RequestConfig,
	params []externalapi.Param,
	paramValues map[string]any,
	secretValues map[string]string,
) (*RenderedRequest, error) {
	values := make(map[string]placeholderValue, len(params))
	for _, param := range params {
		value, ok := paramValues[param.Name]
		values[param.Name] = placeholderValue{
			Value:    value,
			Present:  ok && value != nil,
			Required: param.Required,
		}
	}

	renderedURL, err := renderURLTemplate(request.URLTemplate, values, secretValues)
	if err != nil {
		return nil, err
	}
	renderedHeaders, err := renderHeaderTemplates(request.Headers, values, secretValues)
	if err != nil {
		return nil, err
	}
	renderedBody, _, err := renderBodyTemplate(request.BodyTemplate, values, secretValues)
	if err != nil {
		return nil, err
	}

	return &RenderedRequest{
		URL:     renderedURL,
		Method:  strings.ToUpper(strings.TrimSpace(request.Method)),
		Headers: renderedHeaders,
		Body:    renderedBody,
	}, nil
}

func renderURLTemplate(
	template string,
	values map[string]placeholderValue,
	secrets map[string]string,
) (string, error) {
	queryIndex := strings.Index(template, "?")
	if queryIndex < 0 {
		return renderStringTemplate(template, values, secrets, false)
	}

	base, err := renderStringTemplate(template[:queryIndex], values, secrets, false)
	if err != nil {
		return "", err
	}

	queryRaw := template[queryIndex+1:]
	if strings.TrimSpace(queryRaw) == "" {
		return base, nil
	}

	segments := strings.Split(queryRaw, "&")
	renderedSegments := make([]string, 0, len(segments))
	for _, segment := range segments {
		if strings.TrimSpace(segment) == "" {
			continue
		}
		parts := strings.SplitN(segment, "=", 2)
		key, omit, err := renderQueryComponent(parts[0], values, secrets)
		if err != nil {
			return "", err
		}
		if omit || strings.TrimSpace(key) == "" {
			continue
		}

		if len(parts) == 1 {
			renderedSegments = append(renderedSegments, url.QueryEscape(key))
			continue
		}

		value, omit, err := renderQueryComponent(parts[1], values, secrets)
		if err != nil {
			return "", err
		}
		if omit {
			continue
		}
		renderedSegments = append(
			renderedSegments,
			url.QueryEscape(key)+"="+url.QueryEscape(value),
		)
	}

	if len(renderedSegments) == 0 {
		return base, nil
	}
	return base + "?" + strings.Join(renderedSegments, "&"), nil
}

func renderQueryComponent(
	template string,
	values map[string]placeholderValue,
	secrets map[string]string,
) (string, bool, error) {
	rendered, omit, err := renderStringTemplateMaybeOmit(template, values, secrets)
	if err != nil {
		return "", false, err
	}
	if omit {
		return "", true, nil
	}
	return rendered, false, nil
}

func renderHeaderTemplates(
	headers []externalapi.HeaderTemplate,
	values map[string]placeholderValue,
	secrets map[string]string,
) (map[string]string, error) {
	rendered := make(map[string]string, len(headers))
	for _, header := range headers {
		key, omit, err := renderStringTemplateMaybeOmit(header.KeyTemplate, values, secrets)
		if err != nil {
			return nil, err
		}
		if omit || strings.TrimSpace(key) == "" {
			continue
		}

		value, omit, err := renderStringTemplateMaybeOmit(header.ValueTemplate, values, secrets)
		if err != nil {
			return nil, err
		}
		if omit || strings.TrimSpace(value) == "" {
			continue
		}
		rendered[key] = value
	}
	return rendered, nil
}

func renderBodyTemplate( //nolint:gocognit,cyclop // template rendering
	template any,
	values map[string]placeholderValue,
	secrets map[string]string,
) (any, bool, error) {
	switch typed := template.(type) {
	case nil:
		return nil, false, nil
	case map[string]any:
		result := make(map[string]any, len(typed))
		for rawKey, rawValue := range typed {
			key, omit, err := renderStringTemplateMaybeOmit(rawKey, values, secrets)
			if err != nil {
				return nil, false, err
			}
			if omit || strings.TrimSpace(key) == "" {
				continue
			}

			value, omit, err := renderBodyTemplate(rawValue, values, secrets)
			if err != nil {
				return nil, false, err
			}
			if omit {
				continue
			}
			result[key] = value
		}
		return result, false, nil
	case []any:
		result := make([]any, 0, len(typed))
		for _, rawValue := range typed {
			value, omit, err := renderBodyTemplate(rawValue, values, secrets)
			if err != nil {
				return nil, false, err
			}
			if omit {
				continue
			}
			result = append(result, value)
		}
		return result, false, nil
	case string:
		if match := fullPlaceholderPattern.FindStringSubmatch(typed); len(match) == 2 {
			value, missingOptional, err := resolveTemplateReference(strings.TrimSpace(match[1]), values, secrets)
			if err != nil {
				return nil, false, err
			}
			if missingOptional {
				return nil, true, nil
			}
			return value, false, nil
		}

		rendered, omit, err := renderStringTemplateMaybeOmit(typed, values, secrets)
		if err != nil {
			return nil, false, err
		}
		if omit {
			return nil, true, nil
		}
		return rendered, false, nil
	default:
		return template, false, nil
	}
}

func renderStringTemplateMaybeOmit(
	template string,
	values map[string]placeholderValue,
	secrets map[string]string,
) (string, bool, error) {
	var builder strings.Builder
	lastIndex := 0
	for _, match := range templatePlaceholderPattern.FindAllStringSubmatchIndex(template, -1) {
		if len(match) < 4 {
			continue
		}
		builder.WriteString(template[lastIndex:match[0]])
		ref := strings.TrimSpace(template[match[2]:match[3]])
		value, missingOptional, err := resolveTemplateReference(ref, values, secrets)
		if err != nil {
			return "", false, err
		}
		if missingOptional {
			return "", true, nil
		}
		fmt.Fprint(&builder, value)
		lastIndex = match[1]
	}
	builder.WriteString(template[lastIndex:])
	return builder.String(), false, nil
}

func renderStringTemplate(
	template string,
	values map[string]placeholderValue,
	secrets map[string]string,
	allowOmit bool,
) (string, error) {
	rendered, omit, err := renderStringTemplateMaybeOmit(template, values, secrets)
	if err != nil {
		return "", err
	}
	if omit && !allowOmit {
		return "", fmt.Errorf("missing required value to render %q", template)
	}
	return rendered, nil
}

func resolveTemplateReference(
	ref string,
	values map[string]placeholderValue,
	secrets map[string]string,
) (any, bool, error) {
	if strings.HasPrefix(ref, "secret.") {
		key := strings.TrimPrefix(ref, "secret.")
		value, ok := secrets[key]
		if !ok || strings.TrimSpace(value) == "" {
			return nil, false, fmt.Errorf("missing secret value for %q", key)
		}
		return value, false, nil
	}

	value, ok := values[ref]
	if !ok {
		return nil, false, fmt.Errorf("unknown template parameter %q", ref)
	}
	if !value.Present {
		if value.Required {
			return nil, false, fmt.Errorf("missing required parameter %q", ref)
		}
		return nil, true, nil
	}
	return value.Value, false, nil
}
