package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rendis/feature-evaluator/apps/mcp/internal/client"
)

// --- List Features ---

type ListFeaturesInput struct {
	Search      string `json:"search,omitempty" jsonschema:"filter by name or key"`
	ValueType   string `json:"value_type,omitempty" jsonschema:"filter by value type: boolean, string, number, json"`
	Enabled     string `json:"enabled,omitempty" jsonschema:"filter by enabled: true or false"`
	Tag         string `json:"tag,omitempty" jsonschema:"filter by tag key"`
	Environment string `json:"environment,omitempty" jsonschema:"filter by environment"`
	Page        int    `json:"page,omitempty" jsonschema:"page number (default 1)"`
	PageSize    int    `json:"page_size,omitempty" jsonschema:"items per page (default 20)"`
}

type ListFeaturesOutput struct {
	Result any `json:"result" jsonschema:"paginated list of features"`
}

func listFeatures(c *client.Client) func(context.Context, *mcp.CallToolRequest, ListFeaturesInput) (*mcp.CallToolResult, ListFeaturesOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input ListFeaturesInput) (*mcp.CallToolResult, ListFeaturesOutput, error) {
		q := queryParams(
			"search", input.Search,
			"valueType", input.ValueType,
			"enabled", input.Enabled,
			"tag", input.Tag,
			"environment", input.Environment,
			"page", itoa(input.Page),
			"pageSize", itoa(input.PageSize),
		)
		result, err := c.Get("/features" + q)
		if err != nil {
			return nil, ListFeaturesOutput{}, fmt.Errorf("list features: %w", err)
		}
		return nil, ListFeaturesOutput{Result: result}, nil
	}
}

// --- Get Feature ---

type GetFeatureInput struct {
	Key string `json:"key" jsonschema:"required,feature key"`
}

type GetFeatureOutput struct {
	Result any `json:"result" jsonschema:"feature detail with rules"`
}

func getFeature(c *client.Client) func(context.Context, *mcp.CallToolRequest, GetFeatureInput) (*mcp.CallToolResult, GetFeatureOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input GetFeatureInput) (*mcp.CallToolResult, GetFeatureOutput, error) {
		result, err := c.Get("/features/" + input.Key)
		if err != nil {
			return nil, GetFeatureOutput{}, fmt.Errorf("get feature %q: %w", input.Key, err)
		}
		return nil, GetFeatureOutput{Result: result}, nil
	}
}

// --- Create Feature ---

type CreateFeatureInput struct {
	Key          string   `json:"key" jsonschema:"required,unique feature key (kebab-case)"`
	Name         string   `json:"name" jsonschema:"required,display name"`
	Description  string   `json:"description,omitempty" jsonschema:"optional description"`
	Enabled      bool     `json:"enabled,omitempty" jsonschema:"start enabled (default false)"`
	ValueType    string   `json:"value_type" jsonschema:"required,value type: boolean or string or number or json"`
	DefaultValue any      `json:"default_value" jsonschema:"required,default value when no rule matches"`
	Environments []string `json:"environments,omitempty" jsonschema:"restrict to environments (empty = all)"`
	Tags         []string `json:"tags,omitempty" jsonschema:"tag keys to attach"`
}

type CreateFeatureOutput struct {
	Result any `json:"result" jsonschema:"created feature"`
}

func createFeature(c *client.Client) func(context.Context, *mcp.CallToolRequest, CreateFeatureInput) (*mcp.CallToolResult, CreateFeatureOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input CreateFeatureInput) (*mcp.CallToolResult, CreateFeatureOutput, error) {
		body := map[string]any{
			"key":          input.Key,
			"name":         input.Name,
			"description":  input.Description,
			"enabled":      input.Enabled,
			"valueType":    input.ValueType,
			"defaultValue": input.DefaultValue,
			"environments": input.Environments,
			"tags":         input.Tags,
		}
		result, err := c.Post("/features", body)
		if err != nil {
			return nil, CreateFeatureOutput{}, fmt.Errorf("create feature: %w", err)
		}
		return nil, CreateFeatureOutput{Result: result}, nil
	}
}

// --- Update Feature ---

type UpdateFeatureInput struct {
	Key          string   `json:"key" jsonschema:"required,feature key to update"`
	Name         string   `json:"name" jsonschema:"required,display name"`
	Description  string   `json:"description,omitempty" jsonschema:"description"`
	Enabled      *bool    `json:"enabled,omitempty" jsonschema:"enabled state"`
	ValueType    string   `json:"value_type,omitempty" jsonschema:"value type"`
	DefaultValue any      `json:"default_value,omitempty" jsonschema:"default value"`
	Environments []string `json:"environments,omitempty" jsonschema:"environments"`
	Tags         []string `json:"tags,omitempty" jsonschema:"tags"`
}

type UpdateFeatureOutput struct {
	Result any `json:"result" jsonschema:"updated feature"`
}

func updateFeature(c *client.Client) func(context.Context, *mcp.CallToolRequest, UpdateFeatureInput) (*mcp.CallToolResult, UpdateFeatureOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input UpdateFeatureInput) (*mcp.CallToolResult, UpdateFeatureOutput, error) {
		body := map[string]any{
			"name":         input.Name,
			"description":  input.Description,
			"enabled":      input.Enabled,
			"valueType":    input.ValueType,
			"defaultValue": input.DefaultValue,
			"environments": input.Environments,
			"tags":         input.Tags,
		}
		result, err := c.Put("/features/"+input.Key, body)
		if err != nil {
			return nil, UpdateFeatureOutput{}, fmt.Errorf("update feature %q: %w", input.Key, err)
		}
		return nil, UpdateFeatureOutput{Result: result}, nil
	}
}

// --- Toggle Feature ---

type ToggleFeatureInput struct {
	Key     string `json:"key" jsonschema:"required,feature key"`
	Enabled bool   `json:"enabled" jsonschema:"required,true to enable or false to disable"`
}

type ToggleFeatureOutput struct {
	Result any `json:"result" jsonschema:"toggle result"`
}

func toggleFeature(c *client.Client) func(context.Context, *mcp.CallToolRequest, ToggleFeatureInput) (*mcp.CallToolResult, ToggleFeatureOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input ToggleFeatureInput) (*mcp.CallToolResult, ToggleFeatureOutput, error) {
		result, err := c.Patch("/features/"+input.Key+"/toggle", map[string]any{
			"enabled": input.Enabled,
		})
		if err != nil {
			return nil, ToggleFeatureOutput{}, fmt.Errorf("toggle feature %q: %w", input.Key, err)
		}
		return nil, ToggleFeatureOutput{Result: result}, nil
	}
}

// --- Delete Feature ---

type DeleteFeatureInput struct {
	Key string `json:"key" jsonschema:"required,feature key to delete"`
}

type DeleteFeatureOutput struct {
	Status string `json:"status" jsonschema:"deletion status"`
}

func deleteFeature(c *client.Client) func(context.Context, *mcp.CallToolRequest, DeleteFeatureInput) (*mcp.CallToolResult, DeleteFeatureOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input DeleteFeatureInput) (*mcp.CallToolResult, DeleteFeatureOutput, error) {
		if err := c.Delete("/features/" + input.Key); err != nil {
			return nil, DeleteFeatureOutput{}, fmt.Errorf("delete feature %q: %w", input.Key, err)
		}
		return nil, DeleteFeatureOutput{Status: "deleted"}, nil
	}
}

// RegisterFeatureTools registers feature management tools.
func RegisterFeatureTools(server *mcp.Server, c *client.Client) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_list_features",
		Description: "List feature flags with optional search, type, enabled, tag, and environment filters. Returns paginated results.",
	}, listFeatures(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_get_feature",
		Description: "Get a single feature flag by key, including its rules, tags, and configuration.",
	}, getFeature(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_create_feature",
		Description: "Create a new feature flag. Requires key, name, valueType (boolean/string/number/json), and defaultValue.",
	}, createFeature(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_update_feature",
		Description: "Update an existing feature flag's configuration (name, description, default value, environments, tags).",
	}, updateFeature(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_toggle_feature",
		Description: "Enable or disable a feature flag.",
	}, toggleFeature(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_delete_feature",
		Description: "Delete a feature flag by key. This is irreversible.",
	}, deleteFeature(c))
}
