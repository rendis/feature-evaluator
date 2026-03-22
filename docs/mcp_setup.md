# MCP Setup — mcp-openapi-proxy

This guide explains how to configure the MCP server for the Feature Evaluator API using `mcp-openapi-proxy`.

## What Is mcp-openapi-proxy?

A Go binary that reads the OpenAPI/Swagger spec and auto-generates MCP tools — one per API endpoint. Each tool executes real HTTP calls against the backend, so AI agents can manage feature flags, segments, packs, and everything else directly.

Tool naming pattern: `{prefix}_{method}_{sanitized_path}`

Examples:
- `fe_list_features` — List all features
- `fe_create_feature` — Create a feature
- `fe_get_feature` — Get feature by key
- `fe_evaluate` — Evaluate a single feature
- `fe_bulk_evaluate` — Bulk evaluate features

## Prerequisites

1. **Go 1.21+** installed
2. **Install the binary:**

   ```bash
   go install github.com/rendis/mcp-openapi-proxy/cmd/mcp-openapi-proxy@latest
   ```

3. **Swagger spec generated** (if modifying handlers):

   ```bash
   make swagger
   ```

## Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `MCP_SPEC` | URL or path to the OpenAPI spec | `https://raw.githubusercontent.com/rendis/feature-evaluator/main/server/docs/swagger.yaml` |
| `MCP_BASE_URL` | Base URL for API calls | `http://localhost:8080/features` |
| `MCP_AUTH_TOKEN` | Static auth token (dev mode) | `dev-token` |
| `MCP_TOOL_PREFIX` | Prefix for tool names | `fe` |
| `MCP_OIDC_ISSUER` | OIDC issuer URL (for production auth) | `https://auth.example.com` |
| `MCP_OIDC_CLIENT_ID` | OIDC client ID | `my-client-id` |
| `MCP_EXTRA_HEADERS` | Additional headers (comma-separated `Key:Value`) | `X-Workspace:default` |

## Configuration by Agent

### Claude Code (CLI)

The project includes a `.mcp.json` that configures the MCP server automatically. Just open the project in Claude Code — it auto-detects the config.

To add manually:

```bash
# Project scope (shared via .mcp.json in repo)
claude mcp add feature-evaluator -s project -- mcp-openapi-proxy

# User scope (available in all projects)
claude mcp add feature-evaluator -s user -- mcp-openapi-proxy
```

Set env vars in `.mcp.json` or export them before launching Claude Code.

**Verify installation:**

```bash
claude mcp list
claude mcp get feature-evaluator
```

**Remove if needed:**

```bash
claude mcp remove feature-evaluator
```

---

### OpenAI Codex

Edit `~/.codex/config.toml`:

```toml
[mcp_servers.feature-evaluator]
command = "mcp-openapi-proxy"
args = []

[mcp_servers.feature-evaluator.env]
MCP_SPEC = "https://raw.githubusercontent.com/rendis/feature-evaluator/main/server/docs/swagger.yaml"
MCP_BASE_URL = "http://localhost:8080/features"
MCP_AUTH_TOKEN = "dev-token"
MCP_TOOL_PREFIX = "fe"
```

---

### Gemini CLI

Edit `~/.gemini/settings.json` (global) or `.gemini/settings.json` (project):

```json
{
  "mcpServers": {
    "feature-evaluator": {
      "command": "mcp-openapi-proxy",
      "args": [],
      "env": {
        "MCP_SPEC": "https://raw.githubusercontent.com/rendis/feature-evaluator/main/server/docs/swagger.yaml",
        "MCP_BASE_URL": "http://localhost:8080/features",
        "MCP_AUTH_TOKEN": "dev-token",
        "MCP_TOOL_PREFIX": "fe"
      }
    }
  }
}
```

> **Note:** Restart Gemini CLI after modifying the configuration.

---

## OIDC Authentication

For production backends that require OIDC authentication (instead of a static `MCP_AUTH_TOKEN`):

```bash
# Login (opens browser for OIDC PKCE flow)
mcp-openapi-proxy login

# Check auth status
mcp-openapi-proxy status

# Logout (clear stored tokens)
mcp-openapi-proxy logout
```

Set `MCP_OIDC_ISSUER` and `MCP_OIDC_CLIENT_ID` in your env or `.mcp.json` for OIDC to work. Remove `MCP_AUTH_TOKEN` when using OIDC — the proxy will use the OIDC token automatically.

## Troubleshooting

### Binary Not Found

1. Verify `mcp-openapi-proxy` is in PATH:

   ```bash
   which mcp-openapi-proxy
   ```

2. If missing, install it:

   ```bash
   go install github.com/rendis/mcp-openapi-proxy/cmd/mcp-openapi-proxy@latest
   ```

3. Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is in your `PATH`.

### Tools Not Available

1. For Claude Code, verify with:

   ```bash
   claude mcp list
   ```

2. Ensure the server shows as "Connected"

3. Restart the agent/CLI after configuration changes

### Spec Not Loading

1. If using a URL, verify it resolves:

   ```bash
   curl -s -o /dev/null -w "%{http_code}" "$MCP_SPEC"
   ```

2. If using a local path, verify the file exists:

   ```bash
   ls -la server/docs/swagger.yaml
   ```

3. Regenerate if missing:

   ```bash
   make swagger
   ```

### API Calls Failing

1. Verify the backend is running: `curl http://localhost:8080/features/healthz`
2. Check `MCP_BASE_URL` matches the running server
3. Check `MCP_AUTH_TOKEN` is correct (dev mode) or run `mcp-openapi-proxy status` (OIDC mode)

## References

- [mcp-openapi-proxy](https://github.com/rendis/mcp-openapi-proxy) — The MCP proxy binary
- [Claude Code MCP Docs](https://code.claude.com/docs/en/mcp)
- [OpenAI Codex MCP](https://developers.openai.com/codex/mcp/)
- [Gemini CLI MCP](https://geminicli.com/docs/tools/mcp-server/)
- [Model Context Protocol](https://modelcontextprotocol.io/) — Official MCP specification
