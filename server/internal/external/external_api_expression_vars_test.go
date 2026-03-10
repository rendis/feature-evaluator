package external

import (
	"strings"
	"testing"

	"github.com/rendis/feature-evaluator/internal/domain/externalapi"
)

func TestBuildExternalAPIExpressionVarsIncludesParamsAndManualVariables(t *testing.T) {
	vars, err := buildExternalAPIExpressionVars(
		&externalapi.ExternalAPI{
			Params: []externalapi.Param{
				{Name: "user_id", Type: externalapi.ParamTypeString, Required: true},
			},
			ExpressionVariables: []externalapi.ExpressionVariable{
				{Name: "campus_code", Type: externalapi.ParamTypeString, Required: true},
				{Name: "is_enabled", Type: externalapi.ParamTypeBool, Required: false},
			},
		},
		map[string]any{
			"user_id":     "123",
			"campus_code": "north",
			"ignored":     true,
		},
	)
	if err != nil {
		t.Fatalf("buildExternalAPIExpressionVars() error = %v, want nil", err)
	}

	if len(vars) != 2 {
		t.Fatalf("len(vars) = %d, want 2", len(vars))
	}
	if got := vars["user_id"]; got != "123" {
		t.Fatalf("vars[\"user_id\"] = %#v, want %q", got, "123")
	}
	if got := vars["campus_code"]; got != "north" {
		t.Fatalf("vars[\"campus_code\"] = %#v, want %q", got, "north")
	}
	if _, ok := vars["ignored"]; ok {
		t.Fatal("vars should not include undeclared values")
	}
}

func TestBuildExternalAPIExpressionVarsRejectsMissingRequiredManualVariable(t *testing.T) {
	_, err := buildExternalAPIExpressionVars(
		&externalapi.ExternalAPI{
			ExpressionVariables: []externalapi.ExpressionVariable{
				{Name: "campus_code", Type: externalapi.ParamTypeString, Required: true},
			},
		},
		map[string]any{},
	)
	if err == nil {
		t.Fatal("buildExternalAPIExpressionVars() error = nil, want non-nil")
	}
	if got := err.Error(); !strings.Contains(got, `missing required expression variable "campus_code"`) {
		t.Fatalf("buildExternalAPIExpressionVars() error = %q, want missing variable error", got)
	}
}
