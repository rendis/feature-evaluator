package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rendis/feature-evaluator/apps/mcp/internal/client"
)

// --- List Workspaces ---

type ListWorkspacesInput struct {
	IncludeArchived bool `json:"include_archived,omitempty" jsonschema:"include archived workspaces (default false)"`
}

type ListWorkspacesOutput struct {
	Result any `json:"result" jsonschema:"list of workspaces"`
}

func listWorkspaces(c *client.Client) func(context.Context, *mcp.CallToolRequest, ListWorkspacesInput) (*mcp.CallToolResult, ListWorkspacesOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input ListWorkspacesInput) (*mcp.CallToolResult, ListWorkspacesOutput, error) {
		path := "/workspaces" + queryParams("includeArchived", btoa(input.IncludeArchived))
		result, err := c.Get(path)
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
		result, err := c.Get("/workspaces/" + input.Key)
		if err != nil {
			return nil, SetWorkspaceOutput{}, fmt.Errorf("validate workspace %q: %w", input.Key, err)
		}
		c.SetWorkspace(input.Key)
		return nil, SetWorkspaceOutput{Result: result}, nil
	}
}

// --- Create Workspace ---

type CreateWorkspaceInput struct {
	Key         string `json:"key" jsonschema:"required,unique workspace key (kebab-case)"`
	Name        string `json:"name" jsonschema:"required,display name"`
	Description string `json:"description,omitempty" jsonschema:"optional description"`
}

type CreateWorkspaceOutput struct {
	Result any `json:"result" jsonschema:"created workspace"`
}

func createWorkspace(c *client.Client) func(context.Context, *mcp.CallToolRequest, CreateWorkspaceInput) (*mcp.CallToolResult, CreateWorkspaceOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input CreateWorkspaceInput) (*mcp.CallToolResult, CreateWorkspaceOutput, error) {
		body := map[string]string{
			"key":         input.Key,
			"name":        input.Name,
			"description": input.Description,
		}
		result, err := c.Post("/workspaces", body)
		if err != nil {
			return nil, CreateWorkspaceOutput{}, fmt.Errorf("create workspace: %w", err)
		}
		return nil, CreateWorkspaceOutput{Result: result}, nil
	}
}

// --- Update Workspace ---

type UpdateWorkspaceInput struct {
	Key         string `json:"key" jsonschema:"required,workspace key to update"`
	Name        string `json:"name" jsonschema:"required,display name"`
	Description string `json:"description,omitempty" jsonschema:"description"`
}

type UpdateWorkspaceOutput struct {
	Result any `json:"result" jsonschema:"updated workspace"`
}

func updateWorkspace(c *client.Client) func(context.Context, *mcp.CallToolRequest, UpdateWorkspaceInput) (*mcp.CallToolResult, UpdateWorkspaceOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input UpdateWorkspaceInput) (*mcp.CallToolResult, UpdateWorkspaceOutput, error) {
		body := map[string]string{
			"name":        input.Name,
			"description": input.Description,
		}
		result, err := c.Put("/workspaces/"+input.Key, body)
		if err != nil {
			return nil, UpdateWorkspaceOutput{}, fmt.Errorf("update workspace %q: %w", input.Key, err)
		}
		return nil, UpdateWorkspaceOutput{Result: result}, nil
	}
}

// --- Archive Workspace ---

type ArchiveWorkspaceInput struct {
	Key string `json:"key" jsonschema:"required,workspace key to archive"`
}

type ArchiveWorkspaceOutput struct {
	Result any `json:"result" jsonschema:"archive result"`
}

func archiveWorkspace(c *client.Client) func(context.Context, *mcp.CallToolRequest, ArchiveWorkspaceInput) (*mcp.CallToolResult, ArchiveWorkspaceOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input ArchiveWorkspaceInput) (*mcp.CallToolResult, ArchiveWorkspaceOutput, error) {
		result, err := c.Post("/workspaces/"+input.Key+"/archive", nil)
		if err != nil {
			return nil, ArchiveWorkspaceOutput{}, fmt.Errorf("archive workspace %q: %w", input.Key, err)
		}
		return nil, ArchiveWorkspaceOutput{Result: result}, nil
	}
}

// --- Restore Workspace ---

type RestoreWorkspaceInput struct {
	Key string `json:"key" jsonschema:"required,workspace key to restore"`
}

type RestoreWorkspaceOutput struct {
	Result any `json:"result" jsonschema:"restore result"`
}

func restoreWorkspace(c *client.Client) func(context.Context, *mcp.CallToolRequest, RestoreWorkspaceInput) (*mcp.CallToolResult, RestoreWorkspaceOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input RestoreWorkspaceInput) (*mcp.CallToolResult, RestoreWorkspaceOutput, error) {
		result, err := c.Post("/workspaces/"+input.Key+"/restore", nil)
		if err != nil {
			return nil, RestoreWorkspaceOutput{}, fmt.Errorf("restore workspace %q: %w", input.Key, err)
		}
		return nil, RestoreWorkspaceOutput{Result: result}, nil
	}
}

// RegisterWorkspaceTools registers workspace management tools.
func RegisterWorkspaceTools(server *mcp.Server, c *client.Client) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_list_workspaces",
		Description: "List all accessible workspaces. Optionally include archived. Use this first to discover available workspace keys.",
	}, listWorkspaces(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_set_workspace",
		Description: "Switch the active workspace for all subsequent API calls. Validates the workspace exists before switching.",
	}, setWorkspace(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_create_workspace",
		Description: "Create a new workspace with the given key, name, and optional description. Requires owner role.",
	}, createWorkspace(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_update_workspace",
		Description: "Update a workspace's name and description. Requires owner role.",
	}, updateWorkspace(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_archive_workspace",
		Description: "Archive (soft-delete) a workspace. It can be restored later. Requires owner role.",
	}, archiveWorkspace(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_restore_workspace",
		Description: "Restore a previously archived workspace. Requires owner role.",
	}, restoreWorkspace(c))
}
