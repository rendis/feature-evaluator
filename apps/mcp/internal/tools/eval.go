package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rendis/feature-evaluator/apps/mcp/internal/client"
)

// --- Evaluate ---

type EvaluateInput struct {
	FeatureKey string         `json:"feature_key" jsonschema:"required,feature key to evaluate"`
	Context    map[string]any `json:"context,omitempty" jsonschema:"evaluation context (user, tenant, campus, program, headers, requestBody)"`
}

type EvaluateOutput struct {
	Result any `json:"result" jsonschema:"evaluation result with value and reason"`
}

func evaluate(c *client.Client) func(context.Context, *mcp.CallToolRequest, EvaluateInput) (*mcp.CallToolResult, EvaluateOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input EvaluateInput) (*mcp.CallToolResult, EvaluateOutput, error) {
		body := map[string]any{
			"featureKey": input.FeatureKey,
			"context":    input.Context,
		}
		result, err := c.PostEval("/eval", body)
		if err != nil {
			return nil, EvaluateOutput{}, fmt.Errorf("evaluate %q: %w", input.FeatureKey, err)
		}
		return nil, EvaluateOutput{Result: result}, nil
	}
}

// --- Bulk Evaluate ---

type BulkEvaluateInput struct {
	FeatureKeys []string       `json:"feature_keys" jsonschema:"required,list of feature keys to evaluate"`
	Context     map[string]any `json:"context,omitempty" jsonschema:"shared evaluation context"`
}

type BulkEvaluateOutput struct {
	Result any `json:"result" jsonschema:"bulk evaluation results"`
}

func bulkEvaluate(c *client.Client) func(context.Context, *mcp.CallToolRequest, BulkEvaluateInput) (*mcp.CallToolResult, BulkEvaluateOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input BulkEvaluateInput) (*mcp.CallToolResult, BulkEvaluateOutput, error) {
		body := map[string]any{
			"featureKeys": input.FeatureKeys,
			"context":     input.Context,
		}
		result, err := c.PostEval("/eval/bulk", body)
		if err != nil {
			return nil, BulkEvaluateOutput{}, fmt.Errorf("bulk evaluate: %w", err)
		}
		return nil, BulkEvaluateOutput{Result: result}, nil
	}
}

// RegisterEvalTools registers evaluation tools.
func RegisterEvalTools(server *mcp.Server, c *client.Client) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_evaluate",
		Description: "Evaluate a single feature flag with context. Returns the resolved value and match reason. Context can include user, tenant, campus, program objects and headers/requestBody for input contract.",
	}, evaluate(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_bulk_evaluate",
		Description: "Evaluate multiple feature flags at once with shared context. Returns results for each feature.",
	}, bulkEvaluate(c))
}
