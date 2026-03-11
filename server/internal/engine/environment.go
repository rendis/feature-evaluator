package engine

// ExpressionInputData contains normalized request inputs exposed to expressions.
type ExpressionInputData struct {
	Headers     map[string]any
	RequestBody map[string]any
	Derived     map[string]any
	Sources     map[string]map[string]any
}

// BuildEnv constructs the expression evaluation environment from a namespaced context map.
// Each top-level key in evalContext becomes a top-level variable in the expression environment
// (e.g., context["user"] -> env["user"], context["tenant"] -> env["tenant"]).
// Additionally injects: headers, requestBody, derived, resolved segment sources, authenticated (bool),
// plus custom functions (inSegment, now, dateBefore, dateAfter).
func BuildEnv(
	evalContext map[string]any,
	input ExpressionInputData,
	authenticated bool,
	segmentChecker SegmentChecker,
	externalAPIChecker ExternalAPIChecker,
) map[string]any {
	env := make(map[string]any, len(evalContext)+len(input.Sources)+9)

	// Each namespace becomes a top-level variable
	for k, v := range evalContext {
		env[k] = v
	}

	// Ensure "user" always exists (expressions may reference it)
	if _, ok := env["user"]; !ok {
		env["user"] = map[string]any{}
	}

	if input.Headers == nil {
		input.Headers = map[string]any{}
	}
	if input.RequestBody == nil {
		input.RequestBody = map[string]any{}
	}
	if input.Derived == nil {
		input.Derived = map[string]any{}
	}
	if input.Sources == nil {
		input.Sources = map[string]map[string]any{}
	}

	env["headers"] = input.Headers
	env["requestBody"] = input.RequestBody
	env["derived"] = input.Derived
	env["authenticated"] = authenticated
	input.Derived["authenticated"] = authenticated

	for key, value := range input.Sources {
		if value == nil {
			env[key] = map[string]any{}
			continue
		}
		env[key] = value
	}

	// Add custom functions
	for k, v := range BuiltinFunctions(segmentChecker, externalAPIChecker) {
		env[k] = v
	}

	return env
}
