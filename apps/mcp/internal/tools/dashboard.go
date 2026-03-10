package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rendis/feature-evaluator/apps/mcp/internal/client"
)

// --- Dashboard Stats ---

type DashboardStatsInput struct{}

type DashboardStatsOutput struct {
	Result any `json:"result" jsonschema:"dashboard statistics (feature counts, segment counts, etc.)"`
}

func dashboardStats(c *client.Client) func(context.Context, *mcp.CallToolRequest, DashboardStatsInput) (*mcp.CallToolResult, DashboardStatsOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, _ DashboardStatsInput) (*mcp.CallToolResult, DashboardStatsOutput, error) {
		result, err := c.Get("/dashboard/stats")
		if err != nil {
			return nil, DashboardStatsOutput{}, fmt.Errorf("dashboard stats: %w", err)
		}
		return nil, DashboardStatsOutput{Result: result}, nil
	}
}

// --- List Tags ---

type ListTagsInput struct{}

type ListTagsOutput struct {
	Result any `json:"result" jsonschema:"list of tags"`
}

func listTags(c *client.Client) func(context.Context, *mcp.CallToolRequest, ListTagsInput) (*mcp.CallToolResult, ListTagsOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, _ ListTagsInput) (*mcp.CallToolResult, ListTagsOutput, error) {
		result, err := c.Get("/tags")
		if err != nil {
			return nil, ListTagsOutput{}, fmt.Errorf("list tags: %w", err)
		}
		return nil, ListTagsOutput{Result: result}, nil
	}
}

// --- Audit Errors ---

type AuditErrorsInput struct {
	Page     int    `json:"page,omitempty" jsonschema:"page number"`
	PageSize int    `json:"page_size,omitempty" jsonschema:"items per page"`
	Feature  string `json:"feature,omitempty" jsonschema:"filter by feature key"`
}

type AuditErrorsOutput struct {
	Result any `json:"result" jsonschema:"paginated list of evaluation errors"`
}

func auditErrors(c *client.Client) func(context.Context, *mcp.CallToolRequest, AuditErrorsInput) (*mcp.CallToolResult, AuditErrorsOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input AuditErrorsInput) (*mcp.CallToolResult, AuditErrorsOutput, error) {
		q := queryParams("page", itoa(input.Page), "pageSize", itoa(input.PageSize), "feature", input.Feature)
		result, err := c.Get("/audit/errors" + q)
		if err != nil {
			return nil, AuditErrorsOutput{}, fmt.Errorf("audit errors: %w", err)
		}
		return nil, AuditErrorsOutput{Result: result}, nil
	}
}

// --- Changelog ---

type ChangelogInput struct {
	Page     int    `json:"page,omitempty" jsonschema:"page number"`
	PageSize int    `json:"page_size,omitempty" jsonschema:"items per page"`
	Entity   string `json:"entity,omitempty" jsonschema:"filter by entity type (feature, pack, segment, experiment)"`
}

type ChangelogOutput struct {
	Result any `json:"result" jsonschema:"paginated changelog entries"`
}

func changelog(c *client.Client) func(context.Context, *mcp.CallToolRequest, ChangelogInput) (*mcp.CallToolResult, ChangelogOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input ChangelogInput) (*mcp.CallToolResult, ChangelogOutput, error) {
		q := queryParams("page", itoa(input.Page), "pageSize", itoa(input.PageSize))
		result, err := c.Get("/changelog" + q)
		if err != nil {
			return nil, ChangelogOutput{}, fmt.Errorf("changelog: %w", err)
		}
		return nil, ChangelogOutput{Result: result}, nil
	}
}

// RegisterDashboardTools registers dashboard, tags, audit, and changelog tools.
func RegisterDashboardTools(server *mcp.Server, c *client.Client) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_dashboard_stats",
		Description: "Get dashboard statistics: feature counts, segment counts, recent activity, and error summary.",
	}, dashboardStats(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_list_tags",
		Description: "List all feature tags available in the workspace.",
	}, listTags(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_audit_errors",
		Description: "List recent evaluation errors. Useful for debugging rule expression failures.",
	}, auditErrors(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_changelog",
		Description: "View the changelog of feature, pack, segment, and experiment modifications.",
	}, changelog(c))
}
