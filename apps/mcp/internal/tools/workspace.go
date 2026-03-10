package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rendis/feature-evaluator/apps/mcp/internal/client"
)

// --- List Workspaces ---

type ListWorkspacesInput struct{}

type ListWorkspacesOutput struct {
	Result any `json:"result" jsonschema:"list of workspaces"`
}

func listWorkspaces(c *client.Client) func(context.Context, *mcp.CallToolRequest, ListWorkspacesInput) (*mcp.CallToolResult, ListWorkspacesOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, _ ListWorkspacesInput) (*mcp.CallToolResult, ListWorkspacesOutput, error) {
		result, err := c.Get("/workspaces")
		if err != nil {
			return nil, ListWorkspacesOutput{}, fmt.Errorf("list workspaces: %w", err)
		}
		return nil, ListWorkspacesOutput{Result: result}, nil
	}
}

// --- Set Workspace ---

type SetWorkspaceInput struct {
	Key string `json:"key" jsonschema:"required,workspace key to switch to"`
}

type SetWorkspaceOutput struct {
	Result any `json:"result" jsonschema:"workspace details after switch"`
}

func setWorkspace(c *client.Client) func(context.Context, *mcp.CallToolRequest, SetWorkspaceInput) (*mcp.CallToolResult, SetWorkspaceOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input SetWorkspaceInput) (*mcp.CallToolResult, SetWorkspaceOutput, error) {
		// Validate workspace exists
		result, err := c.Get("/workspaces/" + input.Key)
		if err != nil {
			return nil, SetWorkspaceOutput{}, fmt.Errorf("validate workspace %q: %w", input.Key, err)
		}
		c.SetWorkspace(input.Key)
		return nil, SetWorkspaceOutput{Result: result}, nil
	}
}

// RegisterWorkspaceTools registers workspace management tools.
func RegisterWorkspaceTools(server *mcp.Server, c *client.Client) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_list_workspaces",
		Description: "List all accessible workspaces. Use this first to discover available workspace keys.",
	}, listWorkspaces(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_set_workspace",
		Description: "Switch the active workspace for all subsequent API calls. Validates the workspace exists before switching.",
	}, setWorkspace(c))
}
