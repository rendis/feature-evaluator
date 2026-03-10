package external

import (
	"fmt"

	"github.com/rendis/feature-evaluator/internal/domain/externalapi"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

func buildExternalAPIExpressionVars(
	api *externalapi.ExternalAPI,
	paramValues map[string]any,
) (map[string]any, error) {
	if api == nil {
		return map[string]any{}, nil
	}

	vars := make(map[string]any, len(api.Params)+len(api.ExpressionVariables))

	for _, param := range api.Params {
		if value, ok := paramValues[param.Name]; ok && value != nil {
			vars[param.Name] = value
		}
	}

	for _, variable := range api.ExpressionVariables {
		value, ok := paramValues[variable.Name]
		if ok && value != nil {
			vars[variable.Name] = value
			continue
		}
		if variable.Required {
			return nil, apierror.NewBadRequest(
				fmt.Sprintf("missing required expression variable %q", variable.Name),
				"error.invalidExternalAPIParam",
			)
		}
	}

	return vars, nil
}
