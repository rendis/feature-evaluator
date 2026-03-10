package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rendis/feature-evaluator/apps/mcp/internal/client"
)

// --- List Packs ---

type ListPacksInput struct {
	Search string `json:"search,omitempty" jsonschema:"filter by name or key"`
	Page   int    `json:"page,omitempty" jsonschema:"page number"`
}

type ListPacksOutput struct {
	Result any `json:"result" jsonschema:"paginated list of packs"`
}

func listPacks(c *client.Client) func(context.Context, *mcp.CallToolRequest, ListPacksInput) (*mcp.CallToolResult, ListPacksOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input ListPacksInput) (*mcp.CallToolResult, ListPacksOutput, error) {
		q := queryParams("search", input.Search, "page", itoa(input.Page))
		result, err := c.Get("/packs" + q)
		if err != nil {
			return nil, ListPacksOutput{}, fmt.Errorf("list packs: %w", err)
		}
		return nil, ListPacksOutput{Result: result}, nil
	}
}

// --- Get Pack ---

type GetPackInput struct {
	Key string `json:"key" jsonschema:"required,pack key"`
}

type GetPackOutput struct {
	Result any `json:"result" jsonschema:"pack detail"`
}

func getPack(c *client.Client) func(context.Context, *mcp.CallToolRequest, GetPackInput) (*mcp.CallToolResult, GetPackOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input GetPackInput) (*mcp.CallToolResult, GetPackOutput, error) {
		result, err := c.Get("/packs/" + input.Key)
		if err != nil {
			return nil, GetPackOutput{}, fmt.Errorf("get pack %q: %w", input.Key, err)
		}
		return nil, GetPackOutput{Result: result}, nil
	}
}

// --- Create Pack ---

type CreatePackInput struct {
	Key         string   `json:"key" jsonschema:"required,unique pack key (kebab-case)"`
	Name        string   `json:"name" jsonschema:"required,display name"`
	Description string   `json:"description,omitempty" jsonschema:"optional description"`
	FeatureKeys []string `json:"feature_keys,omitempty" jsonschema:"feature keys to include in the pack"`
	Enabled     bool     `json:"enabled,omitempty" jsonschema:"start enabled (default false)"`
}

type CreatePackOutput struct {
	Result any `json:"result" jsonschema:"created pack"`
}

func createPack(c *client.Client) func(context.Context, *mcp.CallToolRequest, CreatePackInput) (*mcp.CallToolResult, CreatePackOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input CreatePackInput) (*mcp.CallToolResult, CreatePackOutput, error) {
		body := map[string]any{
			"key":         input.Key,
			"name":        input.Name,
			"description": input.Description,
			"featureKeys": input.FeatureKeys,
			"enabled":     input.Enabled,
		}
		result, err := c.Post("/packs", body)
		if err != nil {
			return nil, CreatePackOutput{}, fmt.Errorf("create pack: %w", err)
		}
		return nil, CreatePackOutput{Result: result}, nil
	}
}

// --- Toggle Pack ---

type TogglePackInput struct {
	Key     string `json:"key" jsonschema:"required,pack key"`
	Enabled bool   `json:"enabled" jsonschema:"required,true to enable or false to disable"`
}

type TogglePackOutput struct {
	Result any `json:"result" jsonschema:"toggle result"`
}

func togglePack(c *client.Client) func(context.Context, *mcp.CallToolRequest, TogglePackInput) (*mcp.CallToolResult, TogglePackOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input TogglePackInput) (*mcp.CallToolResult, TogglePackOutput, error) {
		result, err := c.Patch("/packs/"+input.Key+"/toggle", map[string]any{
			"enabled": input.Enabled,
		})
		if err != nil {
			return nil, TogglePackOutput{}, fmt.Errorf("toggle pack %q: %w", input.Key, err)
		}
		return nil, TogglePackOutput{Result: result}, nil
	}
}

// --- Activate Pack ---

type ActivatePackInput struct {
	Key        string `json:"key" jsonschema:"required,pack key"`
	TargetType string `json:"target_type" jsonschema:"required,target type: tenant or campus or program"`
	TargetID   string `json:"target_id" jsonschema:"required,target identifier"`
	ExpiresAt  string `json:"expires_at,omitempty" jsonschema:"optional expiration datetime (ISO 8601)"`
}

type ActivatePackOutput struct {
	Result any `json:"result" jsonschema:"activation result"`
}

func activatePack(c *client.Client) func(context.Context, *mcp.CallToolRequest, ActivatePackInput) (*mcp.CallToolResult, ActivatePackOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input ActivatePackInput) (*mcp.CallToolResult, ActivatePackOutput, error) {
		body := map[string]any{
			"targetType": input.TargetType,
			"targetId":   input.TargetID,
		}
		if input.ExpiresAt != "" {
			body["expiresAt"] = input.ExpiresAt
		}
		result, err := c.Post("/packs/"+input.Key+"/activate", body)
		if err != nil {
			return nil, ActivatePackOutput{}, fmt.Errorf("activate pack %q: %w", input.Key, err)
		}
		return nil, ActivatePackOutput{Result: result}, nil
	}
}

// --- Deactivate Pack ---

type DeactivatePackInput struct {
	Key        string `json:"key" jsonschema:"required,pack key"`
	TargetType string `json:"target_type" jsonschema:"required,target type: tenant or campus or program"`
	TargetID   string `json:"target_id" jsonschema:"required,target identifier"`
}

type DeactivatePackOutput struct {
	Status string `json:"status" jsonschema:"deactivation status"`
}

func deactivatePack(c *client.Client) func(context.Context, *mcp.CallToolRequest, DeactivatePackInput) (*mcp.CallToolResult, DeactivatePackOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input DeactivatePackInput) (*mcp.CallToolResult, DeactivatePackOutput, error) {
		err := c.DeleteWithBody("/packs/"+input.Key+"/activate", map[string]any{
			"targetType": input.TargetType,
			"targetId":   input.TargetID,
		})
		if err != nil {
			return nil, DeactivatePackOutput{}, fmt.Errorf("deactivate pack %q: %w", input.Key, err)
		}
		return nil, DeactivatePackOutput{Status: "deactivated"}, nil
	}
}

// RegisterPackTools registers pack management tools.
func RegisterPackTools(server *mcp.Server, c *client.Client) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_list_packs",
		Description: "List feature packs (bundles of features). Packs can be activated on targets (tenants, campuses, programs).",
	}, listPacks(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_get_pack",
		Description: "Get a pack by key, including its feature keys and metadata.",
	}, getPack(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_create_pack",
		Description: "Create a new feature pack. A pack bundles multiple features for activation on targets.",
	}, createPack(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_toggle_pack",
		Description: "Enable or disable a feature pack.",
	}, togglePack(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_activate_pack",
		Description: "Activate a pack on a target (tenant, campus, or program). When active, the pack's features become accessible to that target.",
	}, activatePack(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_deactivate_pack",
		Description: "Remove a pack activation from a target.",
	}, deactivatePack(c))
}
