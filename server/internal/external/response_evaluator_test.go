package external

import (
	"net/http"
	"testing"

	"github.com/rendis/feature-evaluator/internal/domain/externalapi"
)

func TestEvaluateResponseDefaultsToHTTP2xx(t *testing.T) {
	ok, err := EvaluateResponse("", []byte(`{"allowed":true}`), http.StatusOK, http.Header{})
	if err != nil {
		t.Fatalf("EvaluateResponse() error = %v, want nil", err)
	}
	if !ok {
		t.Fatal("EvaluateResponse() = false, want true")
	}

	blocked, err := EvaluateResponse("", []byte(`{"allowed":false}`), http.StatusForbidden, http.Header{})
	if err != nil {
		t.Fatalf("EvaluateResponse() error = %v, want nil", err)
	}
	if blocked {
		t.Fatal("EvaluateResponse() = true, want false")
	}
}

func TestEvaluateResponseCanInspectHTTPStatusHeadersAndBody(t *testing.T) {
	ok, err := EvaluateResponse(
		`response.status == 403 && response.header["x-decision"] == "deny" && response.body.allowed == false`,
		[]byte(`{"allowed":false}`),
		http.StatusForbidden,
		http.Header{"X-Decision": []string{"deny"}},
	)
	if err != nil {
		t.Fatalf("EvaluateResponse() error = %v, want nil", err)
	}
	if !ok {
		t.Fatal("EvaluateResponse() = false, want true")
	}
}

func TestEvaluateResponseFallsBackToResponseTextForNonJSONBodies(t *testing.T) {
	ok, err := EvaluateResponse(
		`response.status == 200 && response.body == "ok"`,
		[]byte("ok"),
		http.StatusOK,
		http.Header{},
	)
	if err != nil {
		t.Fatalf("EvaluateResponse() error = %v, want nil", err)
	}
	if !ok {
		t.Fatal("EvaluateResponse() = false, want true")
	}
}

func TestEvaluateResponseSupportsContainsHelper(t *testing.T) {
	ok, err := EvaluateResponse(
		`response.status == 200 && response.body contains "authorized"`,
		[]byte("authorized:true"),
		http.StatusOK,
		http.Header{},
	)
	if err != nil {
		t.Fatalf("EvaluateResponse() error = %v, want nil", err)
	}
	if !ok {
		t.Fatal("EvaluateResponse() = false, want true")
	}
}

func TestEvaluateExternalAPIResponseSupportsCombinedHTTPAndBodyValidation(t *testing.T) {
	ok, err := EvaluateExternalAPIResponse(
		externalapi.ResponseValidation{
			Mode: externalapi.ValidationModeBoth,
			HTTP: externalapi.HTTPValidation{
				Mode:  externalapi.HTTPValidationModeStatusCode,
				Codes: []int{200, 204},
			},
			Body: externalapi.BodyValidation{
				Expression: `contains(response.body.user.name, "john") and dateBefore(response.body.inscription_date, now())`,
			},
		},
		[]byte(`{"user":{"name":"john doe"},"inscription_date":"2025-01-01T00:00:00Z"}`),
		http.StatusOK,
		http.Header{},
		nil,
	)
	if err != nil {
		t.Fatalf("EvaluateExternalAPIResponse() error = %v, want nil", err)
	}
	if !ok {
		t.Fatal("EvaluateExternalAPIResponse() = false, want true")
	}
}

func TestEvaluateExternalAPIResponseCanReadVars(t *testing.T) {
	ok, err := EvaluateExternalAPIResponse(
		externalapi.ResponseValidation{
			Mode: externalapi.ValidationModeResponseBody,
			HTTP: externalapi.HTTPValidation{Mode: externalapi.HTTPValidationModeAny2xx},
			Body: externalapi.BodyValidation{
				Expression: `vars.campus_code == "north" and response.body.allowed == true`,
			},
		},
		[]byte(`{"allowed":true}`),
		http.StatusOK,
		http.Header{},
		map[string]any{"campus_code": "north"},
	)
	if err != nil {
		t.Fatalf("EvaluateExternalAPIResponse() error = %v, want nil", err)
	}
	if !ok {
		t.Fatal("EvaluateExternalAPIResponse() = false, want true")
	}
}

func TestEvaluateExternalAPIResponseDetailedSupportsHTTPOnlyMode(t *testing.T) {
	details, err := EvaluateExternalAPIResponseDetailed(
		externalapi.ResponseValidation{
			Mode: externalapi.ValidationModeHTTPCode,
			HTTP: externalapi.HTTPValidation{
				Mode:  externalapi.HTTPValidationModeStatusCode,
				Codes: []int{204},
			},
		},
		nil,
		http.StatusNoContent,
		http.Header{},
		nil,
	)
	if err != nil {
		t.Fatalf("EvaluateExternalAPIResponseDetailed() error = %v, want nil", err)
	}

	if !details.Final.Passed {
		t.Fatal("details.Final.Passed = false, want true")
	}
	if !details.HTTP.Applied || !details.HTTP.Passed {
		t.Fatalf("details.HTTP = %#v, want applied=true passed=true", details.HTTP)
	}
	if details.Expression.Applied {
		t.Fatalf("details.Expression.Applied = true, want false")
	}
}

func TestEvaluateExternalAPIResponseDetailedSupportsBodyOnlyMode(t *testing.T) {
	details, err := EvaluateExternalAPIResponseDetailed(
		externalapi.ResponseValidation{
			Mode: externalapi.ValidationModeResponseBody,
			HTTP: externalapi.HTTPValidation{Mode: externalapi.HTTPValidationModeAny2xx},
			Body: externalapi.BodyValidation{
				Expression: `response.body.allowed == true`,
			},
		},
		[]byte(`{"allowed":true}`),
		http.StatusForbidden,
		http.Header{},
		nil,
	)
	if err != nil {
		t.Fatalf("EvaluateExternalAPIResponseDetailed() error = %v, want nil", err)
	}

	if !details.Final.Passed {
		t.Fatal("details.Final.Passed = false, want true")
	}
	if details.HTTP.Applied {
		t.Fatalf("details.HTTP.Applied = true, want false")
	}
	if !details.Expression.Applied || !details.Expression.Passed {
		t.Fatalf("details.Expression = %#v, want applied=true passed=true", details.Expression)
	}
	if details.Expression.ResolvedExpression != "true == true" {
		t.Fatalf(
			"details.Expression.ResolvedExpression = %q, want %q",
			details.Expression.ResolvedExpression,
			"true == true",
		)
	}
}

func TestEvaluateExternalAPIResponseDetailedCapturesExpressionErrors(t *testing.T) {
	details, err := EvaluateExternalAPIResponseDetailed(
		externalapi.ResponseValidation{
			Mode: externalapi.ValidationModeResponseBody,
			HTTP: externalapi.HTTPValidation{Mode: externalapi.HTTPValidationModeAny2xx},
			Body: externalapi.BodyValidation{
				Expression: `response.body.allowed + 1`,
			},
		},
		[]byte(`{"allowed":true}`),
		http.StatusOK,
		http.Header{},
		nil,
	)
	if err != nil {
		t.Fatalf("EvaluateExternalAPIResponseDetailed() error = %v, want nil", err)
	}

	if details.Final.Passed {
		t.Fatal("details.Final.Passed = true, want false")
	}
	if details.Expression.Error == nil {
		t.Fatal("details.Expression.Error = nil, want runtime error")
	}
	if *details.Expression.Error == "" {
		t.Fatal("details.Expression.Error = empty, want message")
	}
	if details.Expression.Passed {
		t.Fatal("details.Expression.Passed = true, want false")
	}
	if details.Expression.ResolvedExpression != "true + 1" {
		t.Fatalf(
			"details.Expression.ResolvedExpression = %q, want %q",
			details.Expression.ResolvedExpression,
			"true + 1",
		)
	}
}

func TestNormalizeResponseHeadersIncludesLowercaseAliasesForExpressions(t *testing.T) {
	normalized := normalizeResponseHeaders(http.Header{
		"X-Tenant":     []string{"chile"},
		"Content-Type": []string{"application/json"},
	})

	if got := normalized["X-Tenant"]; got != "chile" {
		t.Fatalf("normalized[\"X-Tenant\"] = %v, want %q", got, "chile")
	}
	if got := normalized["x-tenant"]; got != "chile" {
		t.Fatalf("normalized[\"x-tenant\"] = %v, want %q", got, "chile")
	}
	if got := normalized["content-type"]; got != "application/json" {
		t.Fatalf("normalized[\"content-type\"] = %v, want %q", got, "application/json")
	}
}

func TestNormalizePreviewHeadersOmitsDuplicateLowercaseAliases(t *testing.T) {
	normalized := normalizePreviewHeaders(http.Header{
		"X-Tenant":     []string{"chile"},
		"Content-Type": []string{"application/json"},
	})

	if got := normalized["X-Tenant"]; got != "chile" {
		t.Fatalf("normalized[\"X-Tenant\"] = %v, want %q", got, "chile")
	}
	if got := normalized["Content-Type"]; got != "application/json" {
		t.Fatalf("normalized[\"Content-Type\"] = %v, want %q", got, "application/json")
	}
	if _, ok := normalized["x-tenant"]; ok {
		t.Fatal("normalized should not include lowercase alias x-tenant")
	}
	if _, ok := normalized["content-type"]; ok {
		t.Fatal("normalized should not include lowercase alias content-type")
	}
}
