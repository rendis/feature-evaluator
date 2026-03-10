package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rendis/feature-evaluator/apps/mcp/internal/client"
)

// --- Validate Expression ---

type ValidateExpressionInput struct {
	Expression string `json:"expression" jsonschema:"required,expr-lang expression to validate"`
}

type ValidateExpressionOutput struct {
	Result any `json:"result" jsonschema:"validation result with valid boolean and optional error"`
}

func validateExpression(c *client.Client) func(context.Context, *mcp.CallToolRequest, ValidateExpressionInput) (*mcp.CallToolResult, ValidateExpressionOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input ValidateExpressionInput) (*mcp.CallToolResult, ValidateExpressionOutput, error) {
		result, err := c.Post("/expression/validate", map[string]any{
			"expression": input.Expression,
		})
		if err != nil {
			return nil, ValidateExpressionOutput{}, fmt.Errorf("validate expression: %w", err)
		}
		return nil, ValidateExpressionOutput{Result: result}, nil
	}
}

// --- Test Expression ---

type TestExpressionInput struct {
	Expression string         `json:"expression" jsonschema:"required,expr-lang expression to test"`
	Context    map[string]any `json:"context" jsonschema:"required,evaluation context to test against (e.g. {user: {role: 'admin'}})"`
}

type TestExpressionOutput struct {
	Result any `json:"result" jsonschema:"test result with value and matched boolean"`
}

func testExpression(c *client.Client) func(context.Context, *mcp.CallToolRequest, TestExpressionInput) (*mcp.CallToolResult, TestExpressionOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input TestExpressionInput) (*mcp.CallToolResult, TestExpressionOutput, error) {
		result, err := c.Post("/expression/test", map[string]any{
			"expression": input.Expression,
			"context":    input.Context,
		})
		if err != nil {
			return nil, TestExpressionOutput{}, fmt.Errorf("test expression: %w", err)
		}
		return nil, TestExpressionOutput{Result: result}, nil
	}
}

// RegisterExpressionTools registers expression validation and testing tools.
func RegisterExpressionTools(server *mcp.Server, c *client.Client) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_validate_expression",
		Description: "Validate an expr-lang expression for syntax correctness. Use before creating rules to catch errors early.",
	}, validateExpression(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_test_expression",
		Description: "Test an expr-lang expression against a sample context. Returns the evaluation result and whether it matched. Available vars: user.*, tenant.*, campus.*, program.*, authenticated, inSegment(key), now(), dateBefore(), dateAfter().",
	}, testExpression(c))
}
