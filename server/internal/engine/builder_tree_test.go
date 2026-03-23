package engine

import (
	"testing"

	"github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noSource is a convenience for tests that don't need source bindings.
var noSource = feature.SourceBindings{}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func parseTree(t *testing.T, expr string) map[string]any {
	t.Helper()
	result := ParseToBuilderTree(expr, noSource)
	require.NotNil(t, result, "expected non-nil tree for expression: %s", expr)
	require.Equal(t, builderVersion, result["version"])
	return result
}

func rootGroup(t *testing.T, tree map[string]any) map[string]any {
	t.Helper()
	root, ok := tree["root"].(map[string]any)
	require.True(t, ok, "root should be a map")
	require.Equal(t, "group", root["kind"])
	return root
}

func items(t *testing.T, group map[string]any) []any {
	t.Helper()
	items, ok := group["items"].([]any)
	require.True(t, ok, "items should be a slice")
	return items
}

func asMap(t *testing.T, v any) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	require.True(t, ok, "expected map[string]any, got %T", v)
	return m
}

// ---------------------------------------------------------------------------
// Static conditions — simple comparisons
// ---------------------------------------------------------------------------

func TestStaticCondition_EqualString(t *testing.T) {
	tree := parseTree(t, `derived.email == "chris@tether.education"`)
	root := rootGroup(t, tree)
	its := items(t, root)
	require.Len(t, its, 1)

	cond := asMap(t, its[0])
	assert.Equal(t, "condition", cond["kind"])
	assert.Equal(t, "static", cond["conditionKind"])
	assert.Equal(t, "==", cond["operator"])
	assert.Equal(t, "chris@tether.education", cond["rightLiteral"])

	left := asMap(t, cond["left"])
	assert.Equal(t, "derived", left["category"])
	assert.Equal(t, "derived.email", left["path"])
}

func TestStaticCondition_NotEqualNumber(t *testing.T) {
	tree := parseTree(t, `user.age != 25`)
	root := rootGroup(t, tree)
	its := items(t, root)
	require.Len(t, its, 1)

	cond := asMap(t, its[0])
	assert.Equal(t, "!=", cond["operator"])
	assert.Equal(t, "25", cond["rightLiteral"])
}

func TestStaticCondition_GreaterThan(t *testing.T) {
	tree := parseTree(t, `user.age > 18`)
	cond := asMap(t, items(t, rootGroup(t, tree))[0])
	assert.Equal(t, ">", cond["operator"])
	assert.Equal(t, "18", cond["rightLiteral"])
}

func TestStaticCondition_GreaterOrEqual(t *testing.T) {
	tree := parseTree(t, `user.age >= 21`)
	cond := asMap(t, items(t, rootGroup(t, tree))[0])
	assert.Equal(t, ">=", cond["operator"])
}

func TestStaticCondition_LessThan(t *testing.T) {
	tree := parseTree(t, `user.age < 65`)
	cond := asMap(t, items(t, rootGroup(t, tree))[0])
	assert.Equal(t, "<", cond["operator"])
}

func TestStaticCondition_LessOrEqual(t *testing.T) {
	tree := parseTree(t, `user.age <= 99`)
	cond := asMap(t, items(t, rootGroup(t, tree))[0])
	assert.Equal(t, "<=", cond["operator"])
}

func TestStaticCondition_BooleanTrue(t *testing.T) {
	tree := parseTree(t, `authenticated == true`)
	cond := asMap(t, items(t, rootGroup(t, tree))[0])
	assert.Equal(t, "==", cond["operator"])
	assert.Equal(t, "true", cond["rightLiteral"])
}

func TestStaticCondition_BooleanFalse(t *testing.T) {
	tree := parseTree(t, `authenticated == false`)
	cond := asMap(t, items(t, rootGroup(t, tree))[0])
	assert.Equal(t, "false", cond["rightLiteral"])
}

func TestStaticCondition_Float(t *testing.T) {
	tree := parseTree(t, `user.score > 3.14`)
	cond := asMap(t, items(t, rootGroup(t, tree))[0])
	assert.Equal(t, ">", cond["operator"])
	assert.Equal(t, "3.14", cond["rightLiteral"])
}

// ---------------------------------------------------------------------------
// String operators (BuiltinNode)
// ---------------------------------------------------------------------------

func TestStringOperator_Contains(t *testing.T) {
	tree := parseTree(t, `derived.name contains "john"`)
	cond := asMap(t, items(t, rootGroup(t, tree))[0])
	assert.Equal(t, "contains", cond["operator"])
	assert.Equal(t, "john", cond["rightLiteral"])

	left := asMap(t, cond["left"])
	assert.Equal(t, "derived.name", left["path"])
}

func TestStringOperator_StartsWith(t *testing.T) {
	tree := parseTree(t, `derived.email startsWith "admin"`)
	cond := asMap(t, items(t, rootGroup(t, tree))[0])
	assert.Equal(t, "startsWith", cond["operator"])
	assert.Equal(t, "admin", cond["rightLiteral"])
}

func TestStringOperator_EndsWith(t *testing.T) {
	tree := parseTree(t, `derived.email endsWith ".edu"`)
	cond := asMap(t, items(t, rootGroup(t, tree))[0])
	assert.Equal(t, "endsWith", cond["operator"])
	assert.Equal(t, ".edu", cond["rightLiteral"])
}

// ---------------------------------------------------------------------------
// In / Not In operators
// ---------------------------------------------------------------------------

func TestInOperator(t *testing.T) {
	tree := parseTree(t, `derived.role in ["admin", "editor"]`)
	cond := asMap(t, items(t, rootGroup(t, tree))[0])
	assert.Equal(t, "in", cond["operator"])
	assert.Equal(t, "admin,editor", cond["rightLiteral"])
}

func TestNotInOperator(t *testing.T) {
	tree := parseTree(t, `not (derived.role in ["viewer"])`)
	cond := asMap(t, items(t, rootGroup(t, tree))[0])
	assert.Equal(t, "not in", cond["operator"])
	assert.Equal(t, "viewer", cond["rightLiteral"])
}

// ---------------------------------------------------------------------------
// ExternalApi conditions
// ---------------------------------------------------------------------------

func TestExternalApi_Simple(t *testing.T) {
	tree := parseTree(t, `externalApi("tether_pack_enabled")`)
	cond := asMap(t, items(t, rootGroup(t, tree))[0])
	assert.Equal(t, "externalApi", cond["conditionKind"])
	assert.Equal(t, "tether_pack_enabled", cond["externalApiKey"])
	assert.Equal(t, false, cond["negate"])
}

func TestExternalApi_Negated(t *testing.T) {
	tree := parseTree(t, `!externalApi("tether_pack_enabled")`)
	cond := asMap(t, items(t, rootGroup(t, tree))[0])
	assert.Equal(t, "externalApi", cond["conditionKind"])
	assert.Equal(t, "tether_pack_enabled", cond["externalApiKey"])
	assert.Equal(t, true, cond["negate"])
}

// ---------------------------------------------------------------------------
// Segment conditions (with SourceBindings)
// ---------------------------------------------------------------------------

func TestSegmentCondition_FieldOp(t *testing.T) {
	sb := feature.SourceBindings{
		Segments: []feature.SegmentSourceBinding{
			{SegmentKey: "premium_users", LookupPath: "user.id"},
		},
	}
	tree := ParseToBuilderTree(`premium_users.tier == "gold"`, sb)
	require.NotNil(t, tree)

	root := rootGroup(t, tree)
	its := items(t, root)
	require.Len(t, its, 1)

	cond := asMap(t, its[0])
	assert.Equal(t, "segment", cond["conditionKind"])
	assert.Equal(t, "premium_users", cond["segmentKey"])

	lookupRef := asMap(t, cond["lookupInputRef"])
	assert.Equal(t, "user.id", lookupRef["path"])

	fieldOps, ok := cond["fieldOps"].([]any)
	require.True(t, ok)
	require.Len(t, fieldOps, 1)

	op := asMap(t, fieldOps[0])
	assert.Equal(t, "tier", op["fieldPath"])
	assert.Equal(t, "==", op["operator"])
	assert.Equal(t, "gold", op["rightLiteral"])
}

// ---------------------------------------------------------------------------
// Logical groups (AND / OR)
// ---------------------------------------------------------------------------

func TestGroupAND_TwoItems(t *testing.T) {
	tree := parseTree(t, `a == "1" && b == "2"`)
	root := rootGroup(t, tree)
	assert.Equal(t, "and", root["connector"])
	assert.Len(t, items(t, root), 2)
}

func TestGroupOR_TwoItems(t *testing.T) {
	tree := parseTree(t, `a == "1" || b == "2"`)
	root := rootGroup(t, tree)
	assert.Equal(t, "or", root["connector"])
	assert.Len(t, items(t, root), 2)
}

func TestGroupAND_ThreeItems_Flattened(t *testing.T) {
	tree := parseTree(t, `a == "1" && b == "2" && c == "3"`)
	root := rootGroup(t, tree)
	assert.Equal(t, "and", root["connector"])
	assert.Len(t, items(t, root), 3)
}

func TestGroupOR_ThreeItems_Flattened(t *testing.T) {
	tree := parseTree(t, `a == "1" || b == "2" || c == "3"`)
	root := rootGroup(t, tree)
	assert.Equal(t, "or", root["connector"])
	assert.Len(t, items(t, root), 3)
}

// ---------------------------------------------------------------------------
// Nested groups
// ---------------------------------------------------------------------------

func TestNestedGroups_ORInsideAND(t *testing.T) {
	tree := parseTree(t, `(a == "1" || b == "2") && c == "3"`)
	root := rootGroup(t, tree)
	assert.Equal(t, "and", root["connector"])

	its := items(t, root)
	require.Len(t, its, 2)

	// First item: nested OR group
	nested := asMap(t, its[0])
	assert.Equal(t, "group", nested["kind"])
	assert.Equal(t, "or", nested["connector"])
	assert.Len(t, items(t, nested), 2)

	// Second item: static condition
	cond := asMap(t, its[1])
	assert.Equal(t, "static", cond["conditionKind"])
}

func TestNestedGroups_ANDInsideOR(t *testing.T) {
	tree := parseTree(t, `a == "1" && (b == "2" || c == "3")`)
	root := rootGroup(t, tree)
	assert.Equal(t, "and", root["connector"])

	its := items(t, root)
	require.Len(t, its, 2)

	cond := asMap(t, its[0])
	assert.Equal(t, "static", cond["conditionKind"])

	nested := asMap(t, its[1])
	assert.Equal(t, "group", nested["kind"])
	assert.Equal(t, "or", nested["connector"])
}

func TestNestedGroups_TwoANDInsideOR(t *testing.T) {
	tree := parseTree(t, `(a == "1" && b == "2") || (c == "3" && d == "4")`)
	root := rootGroup(t, tree)
	assert.Equal(t, "or", root["connector"])

	its := items(t, root)
	require.Len(t, its, 2)

	g1 := asMap(t, its[0])
	assert.Equal(t, "group", g1["kind"])
	assert.Equal(t, "and", g1["connector"])
	assert.Len(t, items(t, g1), 2)

	g2 := asMap(t, its[1])
	assert.Equal(t, "group", g2["kind"])
	assert.Equal(t, "and", g2["connector"])
	assert.Len(t, items(t, g2), 2)
}

// ---------------------------------------------------------------------------
// Mixed expressions
// ---------------------------------------------------------------------------

func TestMixed_StaticAndExternalApi(t *testing.T) {
	tree := parseTree(t, `derived.email == "x" && externalApi("y")`)
	root := rootGroup(t, tree)
	assert.Equal(t, "and", root["connector"])

	its := items(t, root)
	require.Len(t, its, 2)

	assert.Equal(t, "static", asMap(t, its[0])["conditionKind"])
	assert.Equal(t, "externalApi", asMap(t, its[1])["conditionKind"])
}

func TestMixed_StaticOrExternalApi(t *testing.T) {
	tree := parseTree(t, `derived.role == "admin" || externalApi("bypass")`)
	root := rootGroup(t, tree)
	assert.Equal(t, "or", root["connector"])
	assert.Len(t, items(t, root), 2)
}

func TestMixed_ThreeConditions(t *testing.T) {
	tree := parseTree(t, `derived.email == "x" && !externalApi("y") && authenticated == true`)
	root := rootGroup(t, tree)
	assert.Equal(t, "and", root["connector"])

	its := items(t, root)
	require.Len(t, its, 3)

	assert.Equal(t, "static", asMap(t, its[0])["conditionKind"])
	assert.Equal(t, "externalApi", asMap(t, its[1])["conditionKind"])
	assert.Equal(t, true, asMap(t, its[1])["negate"])
	assert.Equal(t, "static", asMap(t, its[2])["conditionKind"])
}

// ---------------------------------------------------------------------------
// Category inference
// ---------------------------------------------------------------------------

func TestCategory_Headers(t *testing.T) {
	tree := parseTree(t, `headers.Authorization == "Bearer x"`)
	cond := asMap(t, items(t, rootGroup(t, tree))[0])
	left := asMap(t, cond["left"])
	assert.Equal(t, "headers", left["category"])
	assert.Equal(t, "headers.Authorization", left["path"])
}

func TestCategory_RequestBody(t *testing.T) {
	tree := parseTree(t, `requestBody.email == "x"`)
	cond := asMap(t, items(t, rootGroup(t, tree))[0])
	left := asMap(t, cond["left"])
	assert.Equal(t, "requestBody", left["category"])
}

func TestCategory_Derived(t *testing.T) {
	tree := parseTree(t, `derived.email == "x"`)
	cond := asMap(t, items(t, rootGroup(t, tree))[0])
	left := asMap(t, cond["left"])
	assert.Equal(t, "derived", left["category"])
}

func TestCategory_UserNamespace(t *testing.T) {
	tree := parseTree(t, `user.name == "x"`)
	cond := asMap(t, items(t, rootGroup(t, tree))[0])
	left := asMap(t, cond["left"])
	assert.Equal(t, "derived", left["category"])
}

func TestCategory_TenantNamespace(t *testing.T) {
	tree := parseTree(t, `tenant.key == "x"`)
	cond := asMap(t, items(t, rootGroup(t, tree))[0])
	left := asMap(t, cond["left"])
	assert.Equal(t, "derived", left["category"])
}

// ---------------------------------------------------------------------------
// Type inference
// ---------------------------------------------------------------------------

func TestTypeInference_String(t *testing.T) {
	tree := parseTree(t, `derived.name == "hello"`)
	cond := asMap(t, items(t, rootGroup(t, tree))[0])
	left := asMap(t, cond["left"])
	// Default type is string since we can only infer from the input ref defaults.
	assert.Equal(t, "string", left["type"])
}

// ---------------------------------------------------------------------------
// Edge cases / fallback
// ---------------------------------------------------------------------------

func TestEdge_EmptyExpression(t *testing.T) {
	result := ParseToBuilderTree("", noSource)
	assert.Nil(t, result)
}

func TestEdge_WhitespaceOnly(t *testing.T) {
	result := ParseToBuilderTree("   ", noSource)
	assert.Nil(t, result)
}

func TestEdge_SyntaxError(t *testing.T) {
	result := ParseToBuilderTree("a == && b", noSource)
	assert.Nil(t, result)
}

func TestEdge_SingleCondition_WrappedInGroup(t *testing.T) {
	tree := parseTree(t, `derived.email == "x"`)
	root := rootGroup(t, tree)
	assert.Equal(t, "group", root["kind"])
	assert.Equal(t, "and", root["connector"])
	assert.Len(t, items(t, root), 1)
}

func TestEdge_UnsupportedFunction_Now(t *testing.T) {
	result := ParseToBuilderTree(`now()`, noSource)
	assert.Nil(t, result)
}

func TestEdge_UnsupportedFunction_DateBefore(t *testing.T) {
	result := ParseToBuilderTree(`dateBefore(user.createdAt, "2024-01-01")`, noSource)
	assert.Nil(t, result)
}

func TestEdge_InSegment_Standalone(t *testing.T) {
	// inSegment() alone is not a complete builder condition.
	result := ParseToBuilderTree(`inSegment("users")`, noSource)
	assert.Nil(t, result)
}

func TestEdge_BareIdentifier_Authenticated(t *testing.T) {
	tree := parseTree(t, `authenticated`)
	cond := asMap(t, items(t, rootGroup(t, tree))[0])
	assert.Equal(t, "static", cond["conditionKind"])
	assert.Equal(t, "==", cond["operator"])
	assert.Equal(t, "true", cond["rightLiteral"])

	left := asMap(t, cond["left"])
	assert.Equal(t, "authenticated", left["path"])
}

func TestEdge_FieldToFieldComparison(t *testing.T) {
	// Field-to-field (no literal on right) is not supported by the builder.
	result := ParseToBuilderTree(`user.age == tenant.maxAge`, noSource)
	assert.Nil(t, result)
}

func TestEdge_MatchesOperator(t *testing.T) {
	tree := parseTree(t, `derived.email matches ".*@tether\\.edu"`)
	cond := asMap(t, items(t, rootGroup(t, tree))[0])
	assert.Equal(t, "matches", cond["operator"])
	assert.Equal(t, `.*@tether\.edu`, cond["rightLiteral"])
}

// ---------------------------------------------------------------------------
// Structure validation: all nodes have required fields
// ---------------------------------------------------------------------------

func TestStructure_AllNodesHaveID(t *testing.T) {
	tree := parseTree(t, `a == "1" && (b == "2" || externalApi("c"))`)
	root := rootGroup(t, tree)
	assert.NotEmpty(t, root["id"])

	for _, item := range items(t, root) {
		m := asMap(t, item)
		assert.NotEmpty(t, m["id"], "node missing id: %v", m)
		assert.NotEmpty(t, m["kind"], "node missing kind: %v", m)
	}
}

func TestStructure_VersionIsSet(t *testing.T) {
	tree := parseTree(t, `a == "1"`)
	assert.Equal(t, builderVersion, tree["version"])
}

// ---------------------------------------------------------------------------
// Complex real-world expressions
// ---------------------------------------------------------------------------

func TestRealWorld_TetherExpression(t *testing.T) {
	expr := `derived.email == "chris@tether.education" && externalApi("tether_pack_enabled")`
	tree := parseTree(t, expr)
	root := rootGroup(t, tree)
	assert.Equal(t, "and", root["connector"])

	its := items(t, root)
	require.Len(t, its, 2)

	email := asMap(t, its[0])
	assert.Equal(t, "static", email["conditionKind"])
	assert.Equal(t, "==", email["operator"])
	assert.Equal(t, "chris@tether.education", email["rightLiteral"])

	api := asMap(t, its[1])
	assert.Equal(t, "externalApi", api["conditionKind"])
	assert.Equal(t, "tether_pack_enabled", api["externalApiKey"])
}

func TestRealWorld_ComplexWithNegation(t *testing.T) {
	expr := `derived.role == "admin" && !externalApi("maintenance") && user.verified == true`
	tree := parseTree(t, expr)
	root := rootGroup(t, tree)
	assert.Equal(t, "and", root["connector"])
	assert.Len(t, items(t, root), 3)
}

func TestRealWorld_MultipleORsWithNested(t *testing.T) {
	expr := `(derived.role == "admin" || derived.role == "editor") && tenant.key == "acme"`
	tree := parseTree(t, expr)
	root := rootGroup(t, tree)
	assert.Equal(t, "and", root["connector"])

	its := items(t, root)
	require.Len(t, its, 2)

	orGroup := asMap(t, its[0])
	assert.Equal(t, "group", orGroup["kind"])
	assert.Equal(t, "or", orGroup["connector"])
	assert.Len(t, items(t, orGroup), 2)
}

func TestRealWorld_ContainsWithAnd(t *testing.T) {
	expr := `derived.email contains "@tether" && authenticated == true`
	tree := parseTree(t, expr)
	root := rootGroup(t, tree)
	assert.Equal(t, "and", root["connector"])

	its := items(t, root)
	require.Len(t, its, 2)

	containsCond := asMap(t, its[0])
	assert.Equal(t, "contains", containsCond["operator"])
}

func TestRealWorld_SegmentWithStaticCondition(t *testing.T) {
	sb := feature.SourceBindings{
		Segments: []feature.SegmentSourceBinding{
			{SegmentKey: "vip_users", LookupPath: "user.email"},
		},
	}
	expr := `vip_users.plan == "enterprise" && derived.country == "US"`
	tree := ParseToBuilderTree(expr, sb)
	require.NotNil(t, tree)

	root := rootGroup(t, tree)
	assert.Equal(t, "and", root["connector"])

	its := items(t, root)
	require.Len(t, its, 2)

	seg := asMap(t, its[0])
	assert.Equal(t, "segment", seg["conditionKind"])
	assert.Equal(t, "vip_users", seg["segmentKey"])

	static := asMap(t, its[1])
	assert.Equal(t, "static", static["conditionKind"])
	assert.Equal(t, "US", static["rightLiteral"])
}
