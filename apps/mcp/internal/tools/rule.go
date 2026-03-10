package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rendis/feature-evaluator/apps/mcp/internal/client"
)

// --- List Rules ---

type ListRulesInput struct {
	FeatureKey string `json:"feature_key" jsonschema:"required,feature key"`
}

type ListRulesOutput struct {
	Result any `json:"result" jsonschema:"list of rules for the feature"`
}

func listRules(c *client.Client) func(context.Context, *mcp.CallToolRequest, ListRulesInput) (*mcp.CallToolResult, ListRulesOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input ListRulesInput) (*mcp.CallToolResult, ListRulesOutput, error) {
		result, err := c.Get("/features/" + input.FeatureKey + "/rules")
		if err != nil {
			return nil, ListRulesOutput{}, fmt.Errorf("list rules: %w", err)
		}
		return nil, ListRulesOutput{Result: result}, nil
	}
}

// --- Create Rule ---

type CreateRuleInput struct {
	FeatureKey        string `json:"feature_key" jsonschema:"required,feature key to add rule to"`
	Name              string `json:"name" jsonschema:"required,rule display name"`
	Priority          int    `json:"priority,omitempty" jsonschema:"rule priority (lower = evaluated first)"`
	Enabled           bool   `json:"enabled,omitempty" jsonschema:"start enabled (default false)"`
	Expression        string `json:"expression" jsonschema:"required,expr-lang expression (e.g. user.role == 'admin')"`
	Value             any    `json:"value" jsonschema:"required,value returned when expression matches"`
	RolloutPercentage *int   `json:"rollout_percentage,omitempty" jsonschema:"gradual rollout percentage 0-100"`
}

type CreateRuleOutput struct {
	Result any `json:"result" jsonschema:"created rule"`
}

func createRule(c *client.Client) func(context.Context, *mcp.CallToolRequest, CreateRuleInput) (*mcp.CallToolResult, CreateRuleOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input CreateRuleInput) (*mcp.CallToolResult, CreateRuleOutput, error) {
		body := map[string]any{
			"name":              input.Name,
			"priority":          input.Priority,
			"enabled":           input.Enabled,
			"expression":        input.Expression,
			"value":             input.Value,
			"rolloutPercentage": input.RolloutPercentage,
		}
		result, err := c.Post("/features/"+input.FeatureKey+"/rules", body)
		if err != nil {
			return nil, CreateRuleOutput{}, fmt.Errorf("create rule: %w", err)
		}
		return nil, CreateRuleOutput{Result: result}, nil
	}
}

// --- Update Rule ---

type UpdateRuleInput struct {
	FeatureKey        string `json:"feature_key" jsonschema:"required,feature key"`
	RuleID            string `json:"rule_id" jsonschema:"required,rule ID to update"`
	Name              string `json:"name" jsonschema:"required,rule display name"`
	Priority          int    `json:"priority,omitempty" jsonschema:"rule priority"`
	Enabled           bool   `json:"enabled,omitempty" jsonschema:"enabled state"`
	Expression        string `json:"expression" jsonschema:"required,expr-lang expression"`
	Value             any    `json:"value" jsonschema:"required,value returned when matched"`
	RolloutPercentage *int   `json:"rollout_percentage,omitempty" jsonschema:"rollout percentage 0-100"`
}

type UpdateRuleOutput struct {
	Result any `json:"result" jsonschema:"updated rule"`
}

func updateRule(c *client.Client) func(context.Context, *mcp.CallToolRequest, UpdateRuleInput) (*mcp.CallToolResult, UpdateRuleOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input UpdateRuleInput) (*mcp.CallToolResult, UpdateRuleOutput, error) {
		body := map[string]any{
			"name":              input.Name,
			"priority":          input.Priority,
			"enabled":           input.Enabled,
			"expression":        input.Expression,
			"value":             input.Value,
			"rolloutPercentage": input.RolloutPercentage,
		}
		result, err := c.Put("/features/"+input.FeatureKey+"/rules/"+input.RuleID, body)
		if err != nil {
			return nil, UpdateRuleOutput{}, fmt.Errorf("update rule: %w", err)
		}
		return nil, UpdateRuleOutput{Result: result}, nil
	}
}

// --- Delete Rule ---

type DeleteRuleInput struct {
	FeatureKey string `json:"feature_key" jsonschema:"required,feature key"`
	RuleID     string `json:"rule_id" jsonschema:"required,rule ID to delete"`
}

type DeleteRuleOutput struct {
	Status string `json:"status" jsonschema:"deletion status"`
}

func deleteRule(c *client.Client) func(context.Context, *mcp.CallToolRequest, DeleteRuleInput) (*mcp.CallToolResult, DeleteRuleOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input DeleteRuleInput) (*mcp.CallToolResult, DeleteRuleOutput, error) {
		if err := c.Delete("/features/" + input.FeatureKey + "/rules/" + input.RuleID); err != nil {
			return nil, DeleteRuleOutput{}, fmt.Errorf("delete rule: %w", err)
		}
		return nil, DeleteRuleOutput{Status: "deleted"}, nil
	}
}

// --- Reorder Rules ---

type ReorderRulesInput struct {
	FeatureKey string   `json:"feature_key" jsonschema:"required,feature key"`
	RuleIDs    []string `json:"rule_ids" jsonschema:"required,ordered list of rule IDs"`
}

type ReorderRulesOutput struct {
	Result any `json:"result" jsonschema:"reorder result"`
}

func reorderRules(c *client.Client) func(context.Context, *mcp.CallToolRequest, ReorderRulesInput) (*mcp.CallToolResult, ReorderRulesOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input ReorderRulesInput) (*mcp.CallToolResult, ReorderRulesOutput, error) {
		result, err := c.Put("/features/"+input.FeatureKey+"/rules/reorder", map[string]any{
			"ruleIds": input.RuleIDs,
		})
		if err != nil {
			return nil, ReorderRulesOutput{}, fmt.Errorf("reorder rules: %w", err)
		}
		return nil, ReorderRulesOutput{Result: result}, nil
	}
}

// RegisterRuleTools registers rule management tools.
func RegisterRuleTools(server *mcp.Server, c *client.Client) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_list_rules",
		Description: "List all rules for a feature flag, ordered by priority.",
	}, listRules(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_create_rule",
		Description: "Create a targeting rule for a feature. Rules use expr-lang expressions. Available vars: user.*, tenant.*, campus.*, program.*, authenticated, inSegment(key), now(), dateBefore(), dateAfter().",
	}, createRule(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_update_rule",
		Description: "Update an existing rule's name, expression, value, priority, or rollout percentage.",
	}, updateRule(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_delete_rule",
		Description: "Delete a rule from a feature.",
	}, deleteRule(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_reorder_rules",
		Description: "Reorder rules for a feature by providing an ordered list of rule IDs.",
	}, reorderRules(c))
}
