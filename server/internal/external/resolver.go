package external

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rendis/feature-evaluator/internal/domain/externalapi"
	"github.com/rendis/feature-evaluator/internal/domain/feature"
)

// APIResolver resolves external API bindings during rule evaluation.
type APIResolver struct {
	apiSvc *externalapi.Service
	caller *Caller
}

// NewAPIResolver creates a resolver that calls workspace-level external APIs.
func NewAPIResolver(apiSvc *externalapi.Service, caller *Caller) *APIResolver {
	return &APIResolver{apiSvc: apiSvc, caller: caller}
}

// Resolve calls each bound external API and returns a map of apiKey → passed.
func (r *APIResolver) Resolve(
	ctx context.Context,
	bindings []feature.ExternalAPIBinding,
	env map[string]any,
) map[string]bool {
	results := make(map[string]bool, len(bindings))

	for _, binding := range bindings {
		passed, err := r.resolveOne(ctx, binding, env)
		if err != nil {
			slog.Warn("external api binding call failed",
				"apiKey", binding.ExternalAPIKey,
				"failMode", binding.FailMode,
				"error", err,
			)
			// Fail-open: treat failures as passed; fail-closed: treat as failed
			results[binding.ExternalAPIKey] = binding.FailMode == feature.FailModeOpen
			continue
		}
		results[binding.ExternalAPIKey] = passed
	}

	return results
}

func (r *APIResolver) resolveOne(
	ctx context.Context,
	binding feature.ExternalAPIBinding,
	env map[string]any,
) (bool, error) {
	api, err := r.apiSvc.GetByKey(ctx, binding.ExternalAPIKey)
	if err != nil {
		return false, fmt.Errorf("fetching external api %q: %w", binding.ExternalAPIKey, err)
	}

	if !api.Active {
		return false, fmt.Errorf("external api %q is inactive", binding.ExternalAPIKey)
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
			return false, fmt.Errorf("decrypting secrets for %q: %w", binding.ExternalAPIKey, decErr)
		}
		secretValues = decrypted
	}

	return r.caller.CallExternalAPI(ctx, api, paramValues, secretValues)
}
