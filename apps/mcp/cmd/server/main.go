package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rendis/feature-evaluator/apps/mcp/internal/auth"
	"github.com/rendis/feature-evaluator/apps/mcp/internal/client"
	"github.com/rendis/feature-evaluator/apps/mcp/internal/tools"
)

func main() {
	apiBaseURL := os.Getenv("FE_API_URL")
	if apiBaseURL == "" {
		apiBaseURL = "http://localhost:8080/features"
	}

	workspace := os.Getenv("FE_WORKSPACE")
	if workspace == "" {
		workspace = "default"
	}

	// Subcommand routing
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "login":
			if err := auth.RunLogin(); err != nil {
				log.Fatalf("[feature-evaluator-mcp] login failed: %v", err)
			}
			return
		case "logout":
			if err := auth.RunLogout(); err != nil {
				log.Fatalf("[feature-evaluator-mcp] logout failed: %v", err)
			}
			return
		case "status":
			if err := auth.RunStatus(); err != nil {
				log.Fatalf("[feature-evaluator-mcp] status failed: %v", err)
			}
			return
		}
	}

	// MCP server mode
	tp := resolveTokenProvider()

	httpClient := client.New(apiBaseURL, tp, workspace)

	// Health check: warn if API is unreachable but don't block startup.
	if err := httpClient.Healthy(); err != nil {
		fmt.Fprintf(os.Stderr, "[feature-evaluator-mcp] WARNING: API at %s is unreachable: %v\n", apiBaseURL, err)
		fmt.Fprintf(os.Stderr, "[feature-evaluator-mcp] Tools will fail until the API is available. Start it with: make server\n")
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "feature-evaluator",
		Version: "0.1.0",
	}, nil)

	tools.RegisterAll(server, httpClient)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("[feature-evaluator-mcp] server error: %v", err)
	}
}

func resolveTokenProvider() auth.TokenProvider {
	// Explicit token env var takes priority (dev-token or manually provided).
	if token := os.Getenv("FE_AUTH_TOKEN"); token != "" {
		return auth.NewStaticTokenProvider(token)
	}

	// Try OIDC tokens from local file.
	tp, err := auth.NewOIDCTokenProvider(auth.TokenFilePath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "[feature-evaluator-mcp] WARNING: %v\n", err)
		fmt.Fprintf(os.Stderr, "[feature-evaluator-mcp] Falling back to unauthenticated mode. API calls will likely fail.\n")
		return auth.NewStaticTokenProvider("")
	}
	return tp
}
