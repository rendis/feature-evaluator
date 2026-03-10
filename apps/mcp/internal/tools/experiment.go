package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rendis/feature-evaluator/apps/mcp/internal/client"
)

// --- List Experiments ---

type ListExperimentsInput struct {
	Page     int    `json:"page,omitempty" jsonschema:"page number"`
	PageSize int    `json:"page_size,omitempty" jsonschema:"items per page"`
	Status   string `json:"status,omitempty" jsonschema:"filter by status: draft, running, paused, completed"`
}

type ListExperimentsOutput struct {
	Result any `json:"result" jsonschema:"paginated list of experiments"`
}

func listExperiments(c *client.Client) func(context.Context, *mcp.CallToolRequest, ListExperimentsInput) (*mcp.CallToolResult, ListExperimentsOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input ListExperimentsInput) (*mcp.CallToolResult, ListExperimentsOutput, error) {
		q := queryParams("page", itoa(input.Page), "pageSize", itoa(input.PageSize), "status", input.Status)
		result, err := c.Get("/experiments" + q)
		if err != nil {
			return nil, ListExperimentsOutput{}, fmt.Errorf("list experiments: %w", err)
		}
		return nil, ListExperimentsOutput{Result: result}, nil
	}
}

// --- Get Experiment ---

type GetExperimentInput struct {
	ID string `json:"id" jsonschema:"required,experiment ID"`
}

type GetExperimentOutput struct {
	Result any `json:"result" jsonschema:"experiment detail"`
}

func getExperiment(c *client.Client) func(context.Context, *mcp.CallToolRequest, GetExperimentInput) (*mcp.CallToolResult, GetExperimentOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input GetExperimentInput) (*mcp.CallToolResult, GetExperimentOutput, error) {
		result, err := c.Get("/experiments/" + input.ID)
		if err != nil {
			return nil, GetExperimentOutput{}, fmt.Errorf("get experiment: %w", err)
		}
		return nil, GetExperimentOutput{Result: result}, nil
	}
}

// --- Create Experiment ---

type CreateExperimentInput struct {
	FeatureKey  string `json:"feature_key" jsonschema:"required,feature key to experiment on"`
	Name        string `json:"name" jsonschema:"required,experiment name"`
	Description string `json:"description,omitempty" jsonschema:"optional description"`
	Hypothesis  string `json:"hypothesis,omitempty" jsonschema:"hypothesis being tested"`
}

type CreateExperimentOutput struct {
	Result any `json:"result" jsonschema:"created experiment"`
}

func createExperiment(c *client.Client) func(context.Context, *mcp.CallToolRequest, CreateExperimentInput) (*mcp.CallToolResult, CreateExperimentOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input CreateExperimentInput) (*mcp.CallToolResult, CreateExperimentOutput, error) {
		body := map[string]any{
			"featureKey":  input.FeatureKey,
			"name":        input.Name,
			"description": input.Description,
			"hypothesis":  input.Hypothesis,
		}
		result, err := c.Post("/experiments", body)
		if err != nil {
			return nil, CreateExperimentOutput{}, fmt.Errorf("create experiment: %w", err)
		}
		return nil, CreateExperimentOutput{Result: result}, nil
	}
}

// --- Manage Experiment Lifecycle ---

type ManageExperimentInput struct {
	ID     string `json:"id" jsonschema:"required,experiment ID"`
	Action string `json:"action" jsonschema:"required,lifecycle action: start or pause or complete"`
}

type ManageExperimentOutput struct {
	Result any `json:"result" jsonschema:"experiment after action"`
}

func manageExperiment(c *client.Client) func(context.Context, *mcp.CallToolRequest, ManageExperimentInput) (*mcp.CallToolResult, ManageExperimentOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input ManageExperimentInput) (*mcp.CallToolResult, ManageExperimentOutput, error) {
		switch input.Action {
		case "start", "pause", "complete":
		default:
			return nil, ManageExperimentOutput{}, fmt.Errorf("invalid action %q: must be start, pause, or complete", input.Action)
		}
		result, err := c.Post("/experiments/"+input.ID+"/"+input.Action, nil)
		if err != nil {
			return nil, ManageExperimentOutput{}, fmt.Errorf("%s experiment: %w", input.Action, err)
		}
		return nil, ManageExperimentOutput{Result: result}, nil
	}
}

// RegisterExperimentTools registers experiment management tools.
func RegisterExperimentTools(server *mcp.Server, c *client.Client) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_list_experiments",
		Description: "List A/B experiments with optional status filter (draft, running, paused, completed).",
	}, listExperiments(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_get_experiment",
		Description: "Get experiment details by ID, including variants and metrics.",
	}, getExperiment(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_create_experiment",
		Description: "Create a new A/B experiment on a feature flag. Starts in draft status.",
	}, createExperiment(c))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fe_manage_experiment",
		Description: "Manage experiment lifecycle: start (begin collecting data), pause (temporarily stop), or complete (finalize results).",
	}, manageExperiment(c))
}
