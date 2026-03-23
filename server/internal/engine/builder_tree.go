package engine

import (
	"fmt"
	"strings"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
	"github.com/google/uuid"

	"github.com/rendis/feature-evaluator/internal/domain/feature"
)

const (
	builderVersion = 2
	// ConditionsBuilderMetadataKey stores the visual rule-builder tree in rule metadata.
	ConditionsBuilderMetadataKey = "conditionsBuilder"
)

// ParseToBuilderTree converts an expression string into a builder tree compatible
// with the frontend's guided condition builder. It uses the expr-lang AST parser
// and walks the tree to produce the map structure expected by the frontend.
// Returns nil if the expression is empty, invalid, or contains unsupported constructs.
func ParseToBuilderTree(
	expression string,
	sourceBindings feature.SourceBindings,
) map[string]any {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return nil
	}

	tree, err := parser.Parse(expression)
	if err != nil {
		return nil
	}

	segmentKeys := make(map[string]string) // segmentKey → lookupPath
	for _, sb := range sourceBindings.Segments {
		segmentKeys[sb.SegmentKey] = sb.LookupPath
	}

	ctx := &parseContext{
		segmentKeys: segmentKeys,
		depth:       0,
		maxDepth:    MaxASTDepth,
	}

	root := ctx.nodeToBuilder(tree.Node)
	if root == nil {
		return nil
	}

	// Ensure the root is always a group.
	if kind, _ := root["kind"].(string); kind != "group" {
		root = makeGroup("and", []any{root})
	}

	return map[string]any{
		"version": builderVersion,
		"root":    root,
	}
}

type parseContext struct {
	segmentKeys map[string]string // segmentKey → lookupPath
	depth       int
	maxDepth    int
}

// nodeToBuilder converts an AST node to a builder tree map.
// Returns nil for unsupported nodes.
func (ctx *parseContext) nodeToBuilder(node ast.Node) map[string]any {
	if ctx.depth > ctx.maxDepth {
		return nil
	}
	ctx.depth++
	defer func() { ctx.depth-- }()

	switch n := node.(type) {
	case *ast.BinaryNode:
		return ctx.binaryToBuilder(n)
	case *ast.UnaryNode:
		return ctx.unaryToBuilder(n)
	case *ast.CallNode:
		return ctx.callToBuilder(n)
	case *ast.IdentifierNode:
		// Bare identifier like `authenticated` → treat as `authenticated == true`
		return makeStaticCondition(
			makeInputRef(n.Value),
			"==",
			"true",
		)
	default:
		return nil
	}
}

func (ctx *parseContext) binaryToBuilder(n *ast.BinaryNode) map[string]any {
	switch n.Operator {
	case "&&", "||":
		return ctx.logicalToBuilder(n)
	case "==", "!=", ">", ">=", "<", "<=":
		return ctx.comparisonToBuilder(n)
	case "in":
		return ctx.inOperatorToBuilder(n, false)
	case "contains", "startsWith", "endsWith", "matches":
		return ctx.stringBinaryToBuilder(n)
	default:
		return nil
	}
}

func (ctx *parseContext) logicalToBuilder(n *ast.BinaryNode) map[string]any {
	connector := "and"
	if n.Operator == "||" {
		connector = "or"
	}

	// Flatten consecutive same-operator nodes: a && b && c → group{and, [a, b, c]}
	items := ctx.flattenLogical(n, connector)
	if items == nil {
		return nil
	}

	return makeGroup(connector, items)
}

func (ctx *parseContext) flattenLogical(node ast.Node, connector string) []any {
	bn, ok := node.(*ast.BinaryNode)
	if !ok || (bn.Operator != "&&" && bn.Operator != "||") {
		item := ctx.nodeToBuilder(node)
		if item == nil {
			return nil
		}
		return []any{item}
	}

	thisConnector := "and"
	if bn.Operator == "||" {
		thisConnector = "or"
	}

	if thisConnector != connector {
		// Different connector → nested group
		item := ctx.nodeToBuilder(node)
		if item == nil {
			return nil
		}
		return []any{item}
	}

	left := ctx.flattenLogical(bn.Left, connector)
	if left == nil {
		return nil
	}
	right := ctx.flattenLogical(bn.Right, connector)
	if right == nil {
		return nil
	}

	return append(left, right...)
}

func (ctx *parseContext) comparisonToBuilder(n *ast.BinaryNode) map[string]any {
	leftPath := extractFieldPath(n.Left)
	if leftPath == "" {
		return nil
	}

	// Check if this is a segment field comparison.
	if dot := strings.IndexByte(leftPath, '.'); dot > 0 {
		root := leftPath[:dot]
		if _, isSegment := ctx.segmentKeys[root]; isSegment {
			fieldPath := leftPath[dot+1:]
			rightLiteral := extractLiteral(n.Right)
			if rightLiteral == "" {
				return nil
			}
			return makeSegmentFieldCondition(root, ctx.segmentKeys[root], fieldPath, n.Operator, rightLiteral, inferType(n.Right))
		}
	}

	rightLiteral := extractLiteral(n.Right)
	if rightLiteral == "" {
		// Right side might be a field path (field-to-field comparison) — not supported by builder.
		return nil
	}

	return makeStaticCondition(
		makeInputRef(leftPath),
		n.Operator,
		rightLiteral,
	)
}

func (ctx *parseContext) inOperatorToBuilder(n *ast.BinaryNode, negate bool) map[string]any {
	leftPath := extractFieldPath(n.Left)
	if leftPath == "" {
		return nil
	}

	arr, ok := n.Right.(*ast.ArrayNode)
	if !ok {
		return nil
	}

	elements := make([]string, 0, len(arr.Nodes))
	for _, el := range arr.Nodes {
		lit := extractLiteral(el)
		if lit == "" {
			return nil
		}
		elements = append(elements, lit)
	}

	op := "in"
	if negate {
		op = "not in"
	}

	return makeStaticCondition(
		makeInputRef(leftPath),
		op,
		strings.Join(elements, ","),
	)
}

func (ctx *parseContext) stringBinaryToBuilder(n *ast.BinaryNode) map[string]any {
	leftPath := extractFieldPath(n.Left)
	if leftPath == "" {
		return nil
	}
	rightLiteral := extractLiteral(n.Right)
	if rightLiteral == "" {
		return nil
	}
	return makeStaticCondition(makeInputRef(leftPath), n.Operator, rightLiteral)
}

func (ctx *parseContext) unaryToBuilder(n *ast.UnaryNode) map[string]any {
	if n.Operator != "!" && n.Operator != "not" {
		return nil
	}

	// !externalApi("key")
	if call, ok := n.Node.(*ast.CallNode); ok {
		name := extractCallName(call)
		if name == "externalApi" && len(call.Arguments) == 1 {
			key := extractLiteral(call.Arguments[0])
			if key != "" {
				return makeExternalApiCondition(key, true)
			}
		}
	}

	// not (field in [...]) → "not in" operator
	if bn, ok := n.Node.(*ast.BinaryNode); ok && bn.Operator == "in" {
		return ctx.inOperatorToBuilder(bn, true)
	}

	return nil
}

func (ctx *parseContext) callToBuilder(n *ast.CallNode) map[string]any {
	name := extractCallName(n)
	switch name {
	case "externalApi":
		if len(n.Arguments) == 1 {
			key := extractLiteral(n.Arguments[0])
			if key != "" {
				return makeExternalApiCondition(key, false)
			}
		}
	case "inSegment":
		// inSegment("key") by itself is not a complete condition in the builder.
		// It requires field ops. Return nil to fall back to advanced mode.
		return nil
	}

	// Unsupported function.
	return nil
}

// ---------------------------------------------------------------------------
// AST helpers
// ---------------------------------------------------------------------------

// extractFieldPath resolves a field path from an AST node (e.g., "derived.email").
func extractFieldPath(node ast.Node) string {
	switch n := node.(type) {
	case *ast.IdentifierNode:
		return n.Value
	case *ast.MemberNode:
		base := extractFieldPath(n.Node)
		if base == "" {
			return ""
		}
		if prop, ok := n.Property.(*ast.StringNode); ok {
			return base + "." + prop.Value
		}
		if prop, ok := n.Property.(*ast.IdentifierNode); ok {
			return base + "." + prop.Value
		}
		return ""
	case *ast.ChainNode:
		return extractFieldPath(n.Node)
	default:
		return ""
	}
}

// extractLiteral extracts a literal value as a string from an AST node.
func extractLiteral(node ast.Node) string {
	switch n := node.(type) {
	case *ast.StringNode:
		return n.Value
	case *ast.IntegerNode:
		return fmt.Sprintf("%d", n.Value)
	case *ast.FloatNode:
		return fmt.Sprintf("%g", n.Value)
	case *ast.BoolNode:
		if n.Value {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

// extractCallName resolves the function name from a CallNode.
func extractCallName(n *ast.CallNode) string {
	if id, ok := n.Callee.(*ast.IdentifierNode); ok {
		return id.Value
	}
	return ""
}

// inferType determines the builder field type from a literal AST node.
func inferType(node ast.Node) string {
	switch node.(type) {
	case *ast.IntegerNode, *ast.FloatNode:
		return "number"
	case *ast.BoolNode:
		return "boolean"
	default:
		return "string"
	}
}

// inferCategory determines the builder input category from a field path.
func inferCategory(path string) string {
	if strings.HasPrefix(path, "headers.") || path == "headers" {
		return "headers"
	}
	if strings.HasPrefix(path, "requestBody.") || path == "requestBody" {
		return "requestBody"
	}
	return "derived"
}

// ---------------------------------------------------------------------------
// Builder tree constructors
// ---------------------------------------------------------------------------

func newID() string {
	return uuid.New().String()
}

func makeGroup(connector string, items []any) map[string]any {
	return map[string]any{
		"id":        newID(),
		"kind":      "group",
		"connector": connector,
		"items":     items,
	}
}

func makeStaticCondition(left map[string]any, operator string, rightLiteral string) map[string]any {
	return map[string]any{
		"id":            newID(),
		"kind":          "condition",
		"conditionKind": "static",
		"left":          left,
		"operator":      operator,
		"rightLiteral":  rightLiteral,
	}
}

func makeInputRef(path string) map[string]any {
	return makeInputRefTyped(path, "string")
}

func makeInputRefTyped(path string, fieldType string) map[string]any {
	return map[string]any{
		"refKind":  "input",
		"category": inferCategory(path),
		"path":     path,
		"label":    path,
		"type":     fieldType,
	}
}

func makeExternalApiCondition(key string, negate bool) map[string]any {
	return map[string]any{
		"id":             newID(),
		"kind":           "condition",
		"conditionKind":  "externalApi",
		"externalApiKey": key,
		"paramMappings":  []any{},
		"negate":         negate,
	}
}

func makeSegmentFieldCondition(segmentKey, lookupPath, fieldPath, operator, rightLiteral, fieldType string) map[string]any {
	return map[string]any{
		"id":             newID(),
		"kind":           "condition",
		"conditionKind":  "segment",
		"segmentKey":     segmentKey,
		"lookupInputRef": makeInputRefTyped(lookupPath, "string"),
		"fieldOps": []any{
			map[string]any{
				"fieldPath":     fieldPath,
				"fieldType":     fieldType,
				"operator":      operator,
				"rightMode":     "literal",
				"rightLiteral":  rightLiteral,
				"rightInputRef": nil,
			},
		},
		"fieldOpsConnector": "and",
	}
}
