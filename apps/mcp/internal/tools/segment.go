package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rendis/feature-evaluator/apps/mcp/internal/client"
)

// --- List Segments ---

type ListSegmentsInput struct {
	Search   string `json:"search,omitempty" jsonschema:"filter by name or key"`
	Page     int    `json:"page,omitempty" jsonschema:"page number"`
	PageSize int    `json:"page_size,omitempty" jsonschema:"items per page"`
}

type ListSegmentsOutput struct {
	Result any `json:"result" jsonschema:"paginated list of segments"`
}

func listSegments(c *client.Client) func(context.Context, *mcp.CallToolRequest, ListSegmentsInput) (*mcp.CallToolResult, ListSegmentsOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input ListSegmentsInput) (*mcp.CallToolResult, ListSegmentsOutput, error) {
		q := queryParams("search", input.Search, "page", itoa(input.Page), "pageSize", itoa(input.PageSize))
		result, err := c.Get("/segments" + q)
		if err != nil {
			return nil, ListSegmentsOutput{}, fmt.Errorf("list segments: %w", err)
		}
		return nil, ListSegmentsOutput{Result: result}, nil
	}
}

// --- Get Segment ---

type GetSegmentInput struct {
	Key string `json:"key" jsonschema:"required,segment key"`
}

type GetSegmentOutput struct {
	Result any `json:"result" jsonschema:"segment detail"`
}

func getSegment(c *client.Client) func(context.Context, *mcp.CallToolRequest, GetSegmentInput) (*mcp.CallToolResult, GetSegmentOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input GetSegmentInput) (*mcp.CallToolResult, GetSegmentOutput, error) {
		result, err := c.Get("/segments/" + input.Key)
		if err != nil {
			return nil, GetSegmentOutput{}, fmt.Errorf("get segment %q: %w", input.Key, err)
		}
		return nil, GetSegmentOutput{Result: result}, nil
	}
}

// --- Create Segment ---

type CreateSegmentInput struct {
	Key         string `json:"key" jsonschema:"required,unique segment key (kebab-case)"`
	Name        string `json:"name" jsonschema:"required,display name"`
	Description string `json:"description,omitempty" jsonschema:"optional description"`
}

type CreateSegmentOutput struct {
	Result any `json:"result" jsonschema:"created segment"`
}

func createSegment(c *client.Client) func(context.Context, *mcp.CallToolRequest, CreateSegmentInput) (*mcp.CallToolResult, CreateSegmentOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input CreateSegmentInput) (*mcp.CallToolResult, CreateSegmentOutput, error) {
		body := map[string]any{
			"key":         input.Key,
			"name":        input.Name,
			"description": input.Description,
		}
		result, err := c.Post("/segments", body)
		if err != nil {
			return nil, CreateSegmentOutput{}, fmt.Errorf("create segment: %w", err)
		}
		return nil, CreateSegmentOutput{Result: result}, nil
	}
}

// RegisterSegmentTools registers segment management tools.
func RegisterSegmentTools(server *mcp.Server, c *client.Client) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_list_segments",
		Description: "List user segments. Segments are used in rules via inSegment(key) to target specific user groups.",
	}, listSegments(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_get_segment",
		Description: "Get a segment by key, including record count and schema info.",
	}, getSegment(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_create_segment",
		Description: "Create a new user segment. After creation, import data with CSV or JSON records.",
	}, createSegment(c))
}
