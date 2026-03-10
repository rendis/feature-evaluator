package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rendis/feature-evaluator/apps/mcp/internal/client"
)

// RegisterAll registers all feature-evaluator MCP tools on the server.
func RegisterAll(server *mcp.Server, c *client.Client) {
	RegisterWorkspaceTools(server, c)
	RegisterFeatureTools(server, c)
	RegisterRuleTools(server, c)
	RegisterEvalTools(server, c)
	RegisterPackTools(server, c)
	RegisterTierTools(server, c)
	RegisterSegmentTools(server, c)
	RegisterExpressionTools(server, c)
	RegisterExperimentTools(server, c)
	RegisterDashboardTools(server, c)
}
