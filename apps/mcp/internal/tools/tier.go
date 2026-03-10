package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rendis/feature-evaluator/apps/mcp/internal/client"
)

// --- List Tiers ---

type ListTiersInput struct{}

type ListTiersOutput struct {
	Result any `json:"result" jsonschema:"list of predefined tiers"`
}

func listTiers(c *client.Client) func(context.Context, *mcp.CallToolRequest, ListTiersInput) (*mcp.CallToolResult, ListTiersOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, _ ListTiersInput) (*mcp.CallToolResult, ListTiersOutput, error) {
		result, err := c.Get("/tiers")
		if err != nil {
			return nil, ListTiersOutput{}, fmt.Errorf("list tiers: %w", err)
		}
		return nil, ListTiersOutput{Result: result}, nil
	}
}

// RegisterTierTools registers tier management tools.
func RegisterTierTools(server *mcp.Server, c *client.Client) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_list_tiers",
		Description: "List all 24 predefined tiers (key, name, color, icon, category). Tiers are assigned to packs via tier_key. Categories: entry, growth, advanced, top, special, technical.",
	}, listTiers(c))
}
