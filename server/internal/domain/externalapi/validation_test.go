package externalapi

import (
	"strings"
	"testing"
)

func TestValidateAllowsMissingExpressionVariablesForBackwardsCompatibility(t *testing.T) {
	api := validExternalAPIForValidation()
	api.ExpressionVariables = nil

	if err := Validate(api); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestValidateRejectsExpressionVariableConflictingWithParam(t *testing.T) {
	api := validExternalAPIForValidation()
	api.ExpressionVariables = []ExpressionVariable{
		{Name: "user_id", Type: ParamTypeString, Required: true},
	}

	err := Validate(api)
	if err == nil {
		t.Fatal("Validate() error = nil, want non-nil")
	}
	if got := err.Error(); !strings.Contains(got, `expression variable "user_id" conflicts with a request param`) {
		t.Fatalf("Validate() error = %q, want conflict error", got)
	}
}

func TestValidateRejectsSecretPrefixedExpressionVariables(t *testing.T) {
	api := validExternalAPIForValidation()
	api.ExpressionVariables = []ExpressionVariable{
		{Name: "secret.api_token", Type: ParamTypeString},
	}

	err := Validate(api)
	if err == nil {
		t.Fatal("Validate() error = nil, want non-nil")
	}
	if got := err.Error(); !strings.Contains(got, "expression variables cannot use the secret. prefix") {
		t.Fatalf("Validate() error = %q, want secret prefix error", got)
	}
}

func validExternalAPIForValidation() *ExternalAPI {
	return &ExternalAPI{
		Name: "Eligibility API",
		Request: RequestConfig{
			Method:      "POST",
			URLTemplate: "https://api.example.com/check",
			BodyTemplate: map[string]any{
				"userId": "{{user_id}}",
			},
		},
		Params: []Param{
			{
				Name:      "user_id",
				Type:      ParamTypeString,
				Required:  true,
				Locations: []Location{LocationBody},
			},
		},
		ResponseValidation: ResponseValidation{
			Mode: ValidationModeResponseBody,
			HTTP: HTTPValidation{Mode: HTTPValidationModeAny2xx},
			Body: BodyValidation{
				Expression: "response.body != nil",
			},
		},
	}
}
