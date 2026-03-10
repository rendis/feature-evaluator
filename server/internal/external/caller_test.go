package external

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/rendis/feature-evaluator/internal/domain/externalapi"
)

func TestApplyRenderedRequestHeadersInjectsDefaultUserAgent(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://example.com/check", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v, want nil", err)
	}

	applyRenderedRequestHeaders(req, map[string]string{
		"X-Tenant": "chile",
	}, true)

	if got := req.Header.Get("User-Agent"); got != defaultOutboundUserAgent {
		t.Fatalf("User-Agent = %q, want %q", got, defaultOutboundUserAgent)
	}
	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", got, "application/json")
	}
	if got := req.Header.Get("X-Tenant"); got != "chile" {
		t.Fatalf("X-Tenant = %q, want %q", got, "chile")
	}
}

func TestApplyRenderedRequestHeadersPreservesExplicitUserAgent(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://example.com/check", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v, want nil", err)
	}

	applyRenderedRequestHeaders(req, map[string]string{
		"User-Agent": "custom-client/2.0",
	}, false)

	if got := req.Header.Get("User-Agent"); got != "custom-client/2.0" {
		t.Fatalf("User-Agent = %q, want %q", got, "custom-client/2.0")
	}
}

func TestTestExternalAPIIncludesEvaluationDetails(t *testing.T) {
	caller := NewCaller(nil, nil)
	caller.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body:    io.NopCloser(strings.NewReader(`{"approved":true}`)),
			Request: req,
		}, nil
	})

	result, details, err := caller.TestExternalAPI(
		context.Background(),
		&externalapi.ExternalAPI{
			Request: externalapi.RequestConfig{
				Method:      http.MethodPost,
				URLTemplate: "https://example.com/check",
				BodyTemplate: map[string]any{
					"user_id": "{{user_id}}",
				},
			},
			Params: []externalapi.Param{
				{Name: "user_id", Type: externalapi.ParamTypeString, Required: true, Locations: []externalapi.Location{externalapi.LocationBody}},
			},
			ResponseValidation: externalapi.ResponseValidation{
				Mode: externalapi.ValidationModeBoth,
				HTTP: externalapi.HTTPValidation{Mode: externalapi.HTTPValidationModeAny2xx},
				Body: externalapi.BodyValidation{Expression: `response.body.approved == true`},
			},
		},
		map[string]any{"user_id": "123"},
		nil,
	)
	if err != nil {
		t.Fatalf("TestExternalAPI() error = %v, want nil", err)
	}

	if !result.Passed {
		t.Fatal("result.Passed = false, want true")
	}
	if details == nil {
		t.Fatal("details = nil, want payload")
	}
	if details.Request.Method != http.MethodPost {
		t.Fatalf("details.Request.Method = %q, want %q", details.Request.Method, http.MethodPost)
	}
	if details.Evaluations.Final.Mode != externalapi.ValidationModeBoth {
		t.Fatalf("details.Evaluations.Final.Mode = %q, want %q", details.Evaluations.Final.Mode, externalapi.ValidationModeBoth)
	}
	if !details.Evaluations.HTTP.Applied || !details.Evaluations.Expression.Applied {
		t.Fatalf("details.Evaluations = %#v, want both checks applied", details.Evaluations)
	}
	if details.Evaluations.Expression.Error != nil {
		t.Fatalf("details.Evaluations.Expression.Error = %v, want nil", *details.Evaluations.Expression.Error)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
