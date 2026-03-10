package engine

import (
	"testing"
)

func TestValidateDateFunctionArgs(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{
			name:    "dateBefore with 1 arg should fail",
			expr:    `dateBefore("2030-01-01")`,
			wantErr: true,
		},
		{
			name:    "dateBefore with 2 args should pass",
			expr:    `dateBefore(now(), "2030-01-01")`,
			wantErr: false,
		},
		{
			name:    "dateAfter with 1 arg should fail",
			expr:    `dateAfter(x)`,
			wantErr: true,
		},
		{
			name:    "dateAfter with 2 args should pass",
			expr:    `dateAfter(x, y)`,
			wantErr: false,
		},
		{
			name:    "no date functions should pass",
			expr:    `user.id == "abc"`,
			wantErr: false,
		},
		{
			name:    "nested dateBefore with 2 args should pass",
			expr:    `dateBefore(dateAfter(a, b), c)`,
			wantErr: false,
		},
		{
			name:    "dateBefore with 0 args should fail",
			expr:    `dateBefore()`,
			wantErr: true,
		},
		{
			name:    "dateBefore with 3 args should fail",
			expr:    `dateBefore(a, b, c)`,
			wantErr: true,
		},
		{
			name:    "multiple valid calls should pass",
			expr:    `dateBefore(a, b) && dateAfter(c, d)`,
			wantErr: false,
		},
		{
			name:    "first valid second invalid should fail",
			expr:    `dateBefore(a, b) && dateAfter(c)`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDateFunctionArgs(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDateFunctionArgs(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
			}
		})
	}
}

func TestValidateExpressionDateFunctions(t *testing.T) {
	// Ensure ValidateExpression catches date function arg errors
	err := ValidateExpression(`dateBefore("2030-01-01")`)
	if err == nil {
		t.Error("ValidateExpression should reject dateBefore with 1 arg")
	}

	err = ValidateExpression(`dateBefore(now(), "2030-01-01")`)
	if err != nil {
		t.Errorf("ValidateExpression should accept dateBefore with 2 args, got: %v", err)
	}
}
