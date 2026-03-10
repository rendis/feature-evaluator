package external

import (
	"testing"

	"github.com/rendis/feature-evaluator/internal/domain/externalapi"
)

func TestRenderExternalAPIRequestOmitsOptionalPartsAndKeepsTypedBodyValues(t *testing.T) {
	rendered, err := RenderExternalAPIRequest(
		externalapi.RequestConfig{
			Method:      "POST",
			URLTemplate: "https://api.{{env}}.example.com/v1/users/{{user_id}}?campus={{campus_code}}&optional={{optional_query}}",
			Headers: []externalapi.HeaderTemplate{
				{KeyTemplate: "Authorization", ValueTemplate: "Bearer {{secret.api_token}}"},
				{KeyTemplate: "X-Trace", ValueTemplate: "{{optional_header}}"},
			},
			BodyTemplate: map[string]any{
				"userId":   "{{user_id}}",
				"campus":   "{{campus_code}}",
				"premium":  "{{is_premium}}",
				"nickname": "{{optional_body}}",
			},
		},
		[]externalapi.Param{
			{Name: "env", Type: externalapi.ParamTypeString, Required: true, Locations: []externalapi.Location{externalapi.LocationURL}, URLKind: ptr(externalapi.URLKindDomain)},
			{Name: "user_id", Type: externalapi.ParamTypeNumber, Required: true, Locations: []externalapi.Location{externalapi.LocationURL, externalapi.LocationBody}, URLKind: ptr(externalapi.URLKindPath)},
			{Name: "campus_code", Type: externalapi.ParamTypeString, Required: true, Locations: []externalapi.Location{externalapi.LocationURL, externalapi.LocationBody}, URLKind: ptr(externalapi.URLKindQuery)},
			{Name: "optional_query", Type: externalapi.ParamTypeString, Required: false, Locations: []externalapi.Location{externalapi.LocationURL}, URLKind: ptr(externalapi.URLKindQuery)},
			{Name: "optional_header", Type: externalapi.ParamTypeString, Required: false, Locations: []externalapi.Location{externalapi.LocationHeader}},
			{Name: "is_premium", Type: externalapi.ParamTypeBool, Required: true, Locations: []externalapi.Location{externalapi.LocationBody}},
			{Name: "optional_body", Type: externalapi.ParamTypeString, Required: false, Locations: []externalapi.Location{externalapi.LocationBody}},
		},
		map[string]any{
			"env":         "prod",
			"user_id":     42,
			"campus_code": "scl",
			"is_premium":  true,
		},
		map[string]string{
			"api_token": "top-secret",
		},
	)
	if err != nil {
		t.Fatalf("RenderExternalAPIRequest() error = %v, want nil", err)
	}

	if got, want := rendered.URL, "https://api.prod.example.com/v1/users/42?campus=scl"; got != want {
		t.Fatalf("rendered.URL = %q, want %q", got, want)
	}

	if got := rendered.Headers["Authorization"]; got != "Bearer top-secret" {
		t.Fatalf("Authorization header = %q, want %q", got, "Bearer top-secret")
	}
	if _, ok := rendered.Headers["X-Trace"]; ok {
		t.Fatal("optional header should be omitted")
	}

	body := rendered.Body.(map[string]any)
	if got := body["userId"]; got != 42 {
		t.Fatalf("body.userId = %#v, want 42", got)
	}
	if got := body["premium"]; got != true {
		t.Fatalf("body.premium = %#v, want true", got)
	}
	if _, ok := body["nickname"]; ok {
		t.Fatal("optional body field should be omitted")
	}
}

func TestRenderExternalAPIRequestRejectsMissingRequiredParam(t *testing.T) {
	_, err := RenderExternalAPIRequest(
		externalapi.RequestConfig{
			Method:      "GET",
			URLTemplate: "https://api.example.com/users/{{user_id}}",
		},
		[]externalapi.Param{
			{Name: "user_id", Type: externalapi.ParamTypeString, Required: true, Locations: []externalapi.Location{externalapi.LocationURL}, URLKind: ptr(externalapi.URLKindPath)},
		},
		map[string]any{},
		nil,
	)
	if err == nil {
		t.Fatal("RenderExternalAPIRequest() error = nil, want non-nil")
	}
}

func ptr[T any](value T) *T {
	return &value
}
