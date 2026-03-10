package external

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rendis/feature-evaluator/internal/domain/externalapi"
	"github.com/rendis/feature-evaluator/internal/domain/feature"
)

// ExternalApiResolver resolves external API bindings during rule evaluation.
type ExternalApiResolver struct {
	apiSvc *externalapi.Service
	caller *Caller
}

// NewExternalApiResolver creates a resolver that calls workspace-level external APIs.
func NewExternalApiResolver(apiSvc *externalapi.Service, caller *Caller) *ExternalApiResolver {
	return &ExternalApiResolver{apiSvc: apiSvc, caller: caller}
}

// Resolve calls each bound external API and returns a map of apiKey → passed.
func (r *ExternalApiResolver) Resolve(
	ctx context.Context,
	bindings []feature.ExternalApiBinding,
	env map[string]any,
) map[string]bool {
	results := make(map[string]bool, len(bindings))

	for _, binding := range bindings {
		passed, err := r.resolveOne(ctx, binding, env)
		if err != nil {
			slog.Warn("external api binding call failed",
				"apiKey", binding.ExternalApiKey,
				"failMode", binding.FailMode,
				"error", err,
			)
			// Fail-open: treat failures as passed; fail-closed: treat as failed
			results[binding.ExternalApiKey] = binding.FailMode == feature.FailModeOpen
			continue
		}
		results[binding.ExternalApiKey] = passed
	}

	return results
}

func (r *ExternalApiResolver) resolveOne(
	ctx context.Context,
	binding feature.ExternalApiBinding,
	env map[string]any,
) (bool, error) {
	api, err := r.apiSvc.GetByKey(ctx, binding.ExternalApiKey)
	if err != nil {
		return false, fmt.Errorf("fetching external api %q: %w", binding.ExternalApiKey, err)
	}

	if !api.Active {
		return false, fmt.Errorf("external api %q is inactive", binding.ExternalApiKey)
	}

	// Resolve param values from the eval env using the binding's param mappings
	paramValues := make(map[string]any, len(binding.ParamMappings))
	for _, pm := range binding.ParamMappings {
		switch pm.Mode {
		case "literal":
			paramValues[pm.ParamName] = pm.LiteralValue
		case "input":
			paramValues[pm.ParamName] = ResolveValue(pm.InputPath, env)
		}
	}

	// Decrypt secrets for the API call
	var secretValues map[string]string
	if api.HasSecrets && api.SecretPayloadEncrypted != "" {
		decrypted, decErr := r.apiSvc.DecryptSecrets(ctx, api)
		if decErr != nil {
			return false, fmt.Errorf("decrypting secrets for %q: %w", binding.ExternalApiKey, decErr)
		}
		secretValues = decrypted
	}

	return r.caller.CallExternalAPI(ctx, api, paramValues, secretValues)
}
