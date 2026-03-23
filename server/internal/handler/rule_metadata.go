package handler

import (
	"github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/rendis/feature-evaluator/internal/engine"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

const ruleExpressionBuilderIncompatibleMessageKey = "error.ruleExpressionBuilderIncompatible"

func canonicalizeRuleMetadata(
	metadata map[string]any,
	expression string,
	sourceBindings feature.SourceBindings,
) (map[string]any, error) {
	canonical := make(map[string]any, len(metadata)+1)
	for key, value := range metadata {
		if key == engine.ConditionsBuilderMetadataKey {
			continue
		}
		canonical[key] = value
	}

	tree := engine.ParseToBuilderTree(expression, sourceBindings)
	if tree == nil {
		return nil, apierror.NewBadRequest(
			"expression is not compatible with the visual conditions builder",
			ruleExpressionBuilderIncompatibleMessageKey,
		)
	}

	canonical[engine.ConditionsBuilderMetadataKey] = tree

	return canonical, nil
}
