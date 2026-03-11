package external

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"reflect"
	"regexp"
	"slices"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
	"github.com/rendis/feature-evaluator/internal/domain/externalapi"
	"github.com/rendis/feature-evaluator/internal/engine"
)

var responseExpressionFunctionAliases = map[*regexp.Regexp]string{
	regexp.MustCompile(`\bcontains\s*\(`):   "fe_contains(",
	regexp.MustCompile(`\bstartsWith\s*\(`): "fe_startsWith(",
	regexp.MustCompile(`\bendsWith\s*\(`):   "fe_endsWith(",
}

// APIFinalEvaluation summarizes the final boolean produced by the test flow.
type APIFinalEvaluation struct {
	Mode   externalapi.ValidationMode `json:"mode"`
	Passed bool                       `json:"passed"`
}

// APIHTTPValidationDetails describes the HTTP validation stage for a test run.
type APIHTTPValidationDetails struct {
	Applied       bool                           `json:"applied"`
	Passed        bool                           `json:"passed"`
	Mode          externalapi.HTTPValidationMode `json:"mode"`
	ExpectedCodes []int                          `json:"expectedCodes"`
	ActualStatus  int                            `json:"actualStatus"`
}

// APIExpressionValidationDetails describes the expression validation stage for a test run.
type APIExpressionValidationDetails struct {
	Applied            bool    `json:"applied"`
	Passed             bool    `json:"passed"`
	Expression         string  `json:"expression"`
	ResolvedExpression string  `json:"resolvedExpression,omitempty"`
	Error              *string `json:"error"`
}

// APIEvaluationDetails captures all validation stages executed during a test run.
type APIEvaluationDetails struct {
	Final      APIFinalEvaluation             `json:"final"`
	HTTP       APIHTTPValidationDetails       `json:"http"`
	Expression APIExpressionValidationDetails `json:"expression"`
}

// ValidateResponseExpression checks whether a response-body expression is syntactically valid.
func ValidateResponseExpression(expression string) error {
	if err := engine.ValidateExpression(expression); err != nil {
		return fmt.Errorf("response condition security check failed: %w", err)
	}

	compiledExpression := normalizeResponseExpression(expression)
	responseHeaders := map[string]any{}
	env := map[string]any{
		"response": map[string]any{
			"status": http.StatusOK,
			"header": responseHeaders,
			"body":   map[string]any{},
		},
		"responseText": "",
		"vars":         map[string]any{},
		"http": map[string]any{
			"status":  http.StatusOK,
			"headers": responseHeaders,
		},
	}
	for key, value := range engine.BuiltinFunctions(nil, nil) {
		env[key] = value
	}
	if containsFn, ok := env["contains"]; ok {
		env["fe_contains"] = containsFn
	}
	if startsWithFn, ok := env["startsWith"]; ok {
		env["fe_startsWith"] = startsWithFn
	}
	if endsWithFn, ok := env["endsWith"]; ok {
		env["fe_endsWith"] = endsWithFn
	}
	if _, err := expr.Compile(compiledExpression, expr.Env(env), expr.AllowUndefinedVariables()); err != nil {
		return fmt.Errorf("invalid response condition: %w", err)
	}
	return nil
}

// EvaluateExternalAPIResponse validates the HTTP status and/or body using the external API definition.
func EvaluateExternalAPIResponse(
	validation externalapi.ResponseValidation,
	responseBody []byte,
	httpStatus int,
	headers http.Header,
	vars map[string]any,
) (bool, error) {
	details, err := evaluateExternalAPIResponseDetailed(
		validation,
		responseBody,
		httpStatus,
		headers,
		vars,
		false,
	)
	if err != nil {
		return false, err
	}

	return details.Final.Passed, nil
}

// EvaluateExternalAPIResponseDetailed validates the response and returns per-stage debug details.
func EvaluateExternalAPIResponseDetailed(
	validation externalapi.ResponseValidation,
	responseBody []byte,
	httpStatus int,
	headers http.Header,
	vars map[string]any,
) (APIEvaluationDetails, error) {
	return evaluateExternalAPIResponseDetailed(validation, responseBody, httpStatus, headers, vars, true)
}

func evaluateExternalAPIResponseDetailed(
	validation externalapi.ResponseValidation,
	responseBody []byte,
	httpStatus int,
	headers http.Header,
	vars map[string]any,
	captureExpressionError bool,
) (APIEvaluationDetails, error) {
	details := APIEvaluationDetails{
		Final: APIFinalEvaluation{Mode: validation.Mode},
		HTTP: APIHTTPValidationDetails{
			Applied:       validation.Mode == externalapi.ValidationModeHTTPCode || validation.Mode == externalapi.ValidationModeBoth,
			Mode:          validation.HTTP.Mode,
			ExpectedCodes: cloneExpectedCodes(validation.HTTP.Codes),
			ActualStatus:  httpStatus,
		},
		Expression: APIExpressionValidationDetails{
			Applied:    validation.Mode == externalapi.ValidationModeResponseBody || validation.Mode == externalapi.ValidationModeBoth,
			Expression: validation.Body.Expression,
		},
	}

	httpOK := true
	if details.HTTP.Applied {
		var err error
		httpOK, err = evaluateHTTPValidation(validation.HTTP, httpStatus)
		if err != nil {
			return details, err
		}
		details.HTTP.Passed = httpOK
	}

	bodyOK := true
	if details.Expression.Applied { //nolint:nestif // expression evaluation details
		var err error
		var resolvedExpression string
		bodyOK, resolvedExpression, err = evaluateBodyValidation(
			validation.Body.Expression,
			responseBody,
			httpStatus,
			headers,
			vars,
		)
		details.Expression.ResolvedExpression = resolvedExpression
		if err != nil {
			if !captureExpressionError {
				return details, err
			}

			errMessage := err.Error()
			details.Expression.Error = &errMessage
			details.Expression.Passed = false
			bodyOK = false
		} else {
			details.Expression.Passed = bodyOK
		}
	}

	switch validation.Mode {
	case externalapi.ValidationModeHTTPCode:
		details.Final.Passed = httpOK
	case externalapi.ValidationModeResponseBody:
		details.Final.Passed = bodyOK
	case externalapi.ValidationModeBoth:
		details.Final.Passed = httpOK && bodyOK
	default:
		return details, fmt.Errorf("unsupported response validation mode %q", validation.Mode)
	}

	return details, nil
}

func evaluateHTTPValidation(validation externalapi.HTTPValidation, httpStatus int) (bool, error) {
	switch validation.Mode {
	case externalapi.HTTPValidationModeAny2xx:
		return httpStatus >= 200 && httpStatus < 300, nil
	case externalapi.HTTPValidationModeStatusCode:
		for _, code := range validation.Codes {
			if code == httpStatus {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, fmt.Errorf("unsupported http validation mode %q", validation.Mode)
	}
}

func evaluateBodyValidation(
	expression string,
	responseBody []byte,
	httpStatus int,
	headers http.Header,
	vars map[string]any,
) (bool, string, error) {
	if err := ValidateResponseExpression(expression); err != nil {
		return false, "", err
	}

	env := responseExpressionEnv(responseBody, httpStatus, headers, vars)
	resolvedExpression := resolveExpressionWithValues(expression, env)
	result, err := expr.Eval(normalizeResponseExpression(expression), env)
	if err != nil {
		return false, resolvedExpression, fmt.Errorf("evaluating response condition: %w", err)
	}

	boolResult, ok := result.(bool)
	if !ok {
		return false, resolvedExpression, fmt.Errorf("response condition returned non-boolean: %T", result)
	}
	return boolResult, resolvedExpression, nil
}

func responseExpressionEnv(
	responseBody []byte,
	httpStatus int,
	headers http.Header,
	vars map[string]any,
) map[string]any {
	responseText := string(responseBody)
	var responseBodyValue any
	if trimmed := bytes.TrimSpace(responseBody); len(trimmed) == 0 {
		responseBodyValue = nil
	} else if err := json.Unmarshal(responseBody, &responseBodyValue); err != nil {
		responseBodyValue = responseText
	}
	responseHeaders := normalizeResponseHeaders(headers)
	responseEnvelope := map[string]any{
		"status": httpStatus,
		"header": responseHeaders,
		"body":   responseBodyValue,
	}
	if bodyMap, ok := responseBodyValue.(map[string]any); ok {
		for key, value := range bodyMap {
			if _, reserved := responseEnvelope[key]; reserved {
				continue
			}
			responseEnvelope[key] = value
		}
	}

	env := map[string]any{
		"response":     responseEnvelope,
		"responseText": responseText,
		"vars":         cloneExpressionVars(vars),
		"http": map[string]any{
			"status":  httpStatus,
			"headers": responseHeaders,
		},
	}
	for key, value := range engine.BuiltinFunctions(nil, nil) {
		env[key] = value
	}
	if containsFn, ok := env["contains"]; ok {
		env["fe_contains"] = containsFn
	}
	if startsWithFn, ok := env["startsWith"]; ok {
		env["fe_startsWith"] = startsWithFn
	}
	if endsWithFn, ok := env["endsWith"]; ok {
		env["fe_endsWith"] = endsWithFn
	}
	return env
}

func resolveExpressionWithValues(expression string, env map[string]any) string {
	tree, err := parser.Parse(expression)
	if err != nil {
		return expression
	}

	replacements := collectExpressionReplacements(tree.Node, env)
	if len(replacements) == 0 {
		return expression
	}

	resolved := expression
	for i := len(replacements) - 1; i >= 0; i-- {
		replacement := replacements[i]
		resolved = resolved[:replacement.from] + replacement.value + resolved[replacement.to:]
	}
	return resolved
}

type expressionReplacement struct {
	from  int
	to    int
	value string
}

func collectExpressionReplacements(root ast.Node, env map[string]any) []expressionReplacement {
	candidates := make([]expressionReplacement, 0)
	collectExpressionReplacementCandidates(root, nil, env, &candidates)
	if len(candidates) == 0 {
		return nil
	}

	slices.SortFunc(candidates, func(left, right expressionReplacement) int {
		if left.from != right.from {
			return left.from - right.from
		}
		leftLen := left.to - left.from
		rightLen := right.to - right.from
		return rightLen - leftLen
	})

	selected := make([]expressionReplacement, 0, len(candidates))
	lastTo := -1
	for _, candidate := range candidates {
		if candidate.from < lastTo {
			continue
		}
		selected = append(selected, candidate)
		lastTo = candidate.to
	}

	return selected
}

func collectExpressionReplacementCandidates( //nolint:gocognit,cyclop,funlen // AST traversal
	node ast.Node,
	parent ast.Node,
	env map[string]any,
	candidates *[]expressionReplacement,
) {
	switch n := node.(type) {
	case *ast.UnaryNode:
		collectExpressionReplacementCandidates(n.Node, node, env, candidates)
	case *ast.BinaryNode:
		collectExpressionReplacementCandidates(n.Left, node, env, candidates)
		collectExpressionReplacementCandidates(n.Right, node, env, candidates)
	case *ast.ChainNode:
		collectExpressionReplacementCandidates(n.Node, node, env, candidates)
	case *ast.MemberNode:
		collectExpressionReplacementCandidates(n.Node, node, env, candidates)
		collectExpressionReplacementCandidates(n.Property, node, env, candidates)
	case *ast.SliceNode:
		collectExpressionReplacementCandidates(n.Node, node, env, candidates)
		if n.From != nil {
			collectExpressionReplacementCandidates(n.From, node, env, candidates)
		}
		if n.To != nil {
			collectExpressionReplacementCandidates(n.To, node, env, candidates)
		}
	case *ast.CallNode:
		collectExpressionReplacementCandidates(n.Callee, node, env, candidates)
		for _, argument := range n.Arguments {
			collectExpressionReplacementCandidates(argument, node, env, candidates)
		}
	case *ast.BuiltinNode:
		for _, argument := range n.Arguments {
			collectExpressionReplacementCandidates(argument, node, env, candidates)
		}
	case *ast.PredicateNode:
		collectExpressionReplacementCandidates(n.Node, node, env, candidates)
	case *ast.VariableDeclaratorNode:
		collectExpressionReplacementCandidates(n.Value, node, env, candidates)
		collectExpressionReplacementCandidates(n.Expr, node, env, candidates)
	case *ast.SequenceNode:
		for _, item := range n.Nodes {
			collectExpressionReplacementCandidates(item, node, env, candidates)
		}
	case *ast.ConditionalNode:
		collectExpressionReplacementCandidates(n.Cond, node, env, candidates)
		collectExpressionReplacementCandidates(n.Exp1, node, env, candidates)
		collectExpressionReplacementCandidates(n.Exp2, node, env, candidates)
	case *ast.ArrayNode:
		for _, item := range n.Nodes {
			collectExpressionReplacementCandidates(item, node, env, candidates)
		}
	case *ast.MapNode:
		for _, pair := range n.Pairs {
			collectExpressionReplacementCandidates(pair, node, env, candidates)
		}
	case *ast.PairNode:
		collectExpressionReplacementCandidates(n.Key, node, env, candidates)
		collectExpressionReplacementCandidates(n.Value, node, env, candidates)
	}

	switch typed := node.(type) {
	case *ast.MemberNode:
		appendExpressionReplacement(typed, expressionNodeStart(typed), typed.Location().To, env, candidates)
	case *ast.IdentifierNode:
		if shouldSkipIdentifierReplacement(parent) {
			return
		}
		appendExpressionReplacement(typed, typed.Location().From, typed.Location().To, env, candidates)
	}
}

func shouldSkipIdentifierReplacement(parent ast.Node) bool {
	switch parent.(type) {
	case *ast.MemberNode, *ast.CallNode:
		return true
	default:
		return false
	}
}

func appendExpressionReplacement(
	node ast.Node,
	from int,
	to int,
	env map[string]any,
	candidates *[]expressionReplacement,
) {
	value, ok := resolveExpressionNodeValue(node, env)
	if !ok || !isRenderableExpressionValue(value) {
		return
	}

	*candidates = append(*candidates, expressionReplacement{
		from:  from,
		to:    to,
		value: valueToExpressionNode(value).String(),
	})
}

func expressionNodeStart(node ast.Node) int {
	switch typed := node.(type) {
	case *ast.MemberNode:
		return expressionNodeStart(typed.Node)
	case *ast.ChainNode:
		return expressionNodeStart(typed.Node)
	default:
		return node.Location().From
	}
}

func resolveExpressionNodeValue(node ast.Node, env map[string]any) (any, bool) {
	switch n := node.(type) {
	case *ast.IdentifierNode:
		value, ok := env[n.Value]
		return value, ok
	case *ast.MemberNode:
		target, ok := resolveExpressionNodeValue(n.Node, env)
		if !ok {
			return nil, false
		}

		property, ok := expressionPropertyValue(n.Property, env)
		if !ok {
			return nil, false
		}

		return expressionMemberValue(target, property)
	default:
		return nil, false
	}
}

func expressionPropertyValue(node ast.Node, env map[string]any) (any, bool) {
	switch n := node.(type) {
	case *ast.StringNode:
		return n.Value, true
	case *ast.IntegerNode:
		return n.Value, true
	case *ast.FloatNode:
		return int(n.Value), true
	case *ast.IdentifierNode:
		return resolveExpressionNodeValue(n, env)
	default:
		return nil, false
	}
}

func expressionMemberValue(target any, property any) (any, bool) { //nolint:cyclop // type switch
	targetValue := reflect.ValueOf(target)
	if !targetValue.IsValid() {
		return nil, false
	}

	for targetValue.Kind() == reflect.Interface || targetValue.Kind() == reflect.Pointer {
		if targetValue.IsNil() {
			return nil, false
		}
		targetValue = targetValue.Elem()
	}

	switch targetValue.Kind() {
	case reflect.Map:
		key, ok := convertExpressionMapKey(property, targetValue.Type().Key())
		if !ok {
			return nil, false
		}
		value := targetValue.MapIndex(key)
		if !value.IsValid() {
			return nil, false
		}
		return value.Interface(), true
	case reflect.Slice, reflect.Array:
		index, ok := expressionSliceIndex(property)
		if !ok || index < 0 || index >= targetValue.Len() {
			return nil, false
		}
		return targetValue.Index(index).Interface(), true
	case reflect.Struct:
		propertyName, ok := property.(string)
		if !ok {
			return nil, false
		}
		field := targetValue.FieldByName(propertyName)
		if !field.IsValid() {
			return nil, false
		}
		return field.Interface(), true
	default:
		return nil, false
	}
}

func convertExpressionMapKey(property any, keyType reflect.Type) (reflect.Value, bool) {
	value := reflect.ValueOf(property)
	if !value.IsValid() {
		return reflect.Value{}, false
	}
	if value.Type().AssignableTo(keyType) {
		return value, true
	}
	if value.Type().ConvertibleTo(keyType) {
		return value.Convert(keyType), true
	}
	return reflect.Value{}, false
}

func expressionSliceIndex(property any) (int, bool) {
	switch value := property.(type) {
	case int:
		return value, true
	case int8:
		return int(value), true
	case int16:
		return int(value), true
	case int32:
		return int(value), true
	case int64:
		return int(value), true
	case uint:
		if value > math.MaxInt { //nolint:gosec // bounds check
			return 0, false
		}
		return int(value), true //nolint:gosec // checked above
	case uint8:
		return int(value), true
	case uint16:
		return int(value), true
	case uint32:
		return int(value), true
	case uint64:
		if value > math.MaxInt { //nolint:gosec // bounds check
			return 0, false
		}
		return int(value), true //nolint:gosec // checked above
	default:
		return 0, false
	}
}

func isRenderableExpressionValue(value any) bool {
	if value == nil {
		return true
	}

	valueType := reflect.TypeOf(value)
	return valueType.Kind() != reflect.Func
}

func valueToExpressionNode(value any) ast.Node { //nolint:cyclop // type switch
	switch typed := value.(type) {
	case nil:
		return &ast.NilNode{}
	case bool:
		return &ast.BoolNode{Value: typed}
	case string:
		return &ast.StringNode{Value: typed}
	case int:
		return &ast.IntegerNode{Value: typed}
	case int8:
		return &ast.IntegerNode{Value: int(typed)}
	case int16:
		return &ast.IntegerNode{Value: int(typed)}
	case int32:
		return &ast.IntegerNode{Value: int(typed)}
	case int64:
		return &ast.IntegerNode{Value: int(typed)}
	case uint:
		return &ast.IntegerNode{Value: int(min(typed, math.MaxInt))} //nolint:gosec // clamped
	case uint8:
		return &ast.IntegerNode{Value: int(typed)}
	case uint16:
		return &ast.IntegerNode{Value: int(typed)}
	case uint32:
		return &ast.IntegerNode{Value: int(typed)}
	case uint64:
		return &ast.IntegerNode{Value: int(min(typed, math.MaxInt))} //nolint:gosec // clamped
	case float32:
		return &ast.FloatNode{Value: float64(typed)}
	case float64:
		return &ast.FloatNode{Value: typed}
	default:
		return &ast.ConstantNode{Value: value}
	}
}

func cloneExpressionVars(paramValues map[string]any) map[string]any {
	if len(paramValues) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(paramValues))
	for key, value := range paramValues {
		cloned[key] = value
	}
	return cloned
}

func normalizeResponseExpression(expression string) string {
	normalized := expression
	for pattern, replacement := range responseExpressionFunctionAliases {
		normalized = pattern.ReplaceAllString(normalized, replacement)
	}
	return normalized
}

func cloneExpectedCodes(codes []int) []int {
	if len(codes) == 0 {
		return []int{}
	}

	cloned := make([]int, len(codes))
	copy(cloned, codes)
	return cloned
}
