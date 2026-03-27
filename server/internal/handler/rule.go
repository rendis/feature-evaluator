package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rendis/feature-evaluator/internal/domain/changelog"
	"github.com/rendis/feature-evaluator/internal/domain/evaluation"
	"github.com/rendis/feature-evaluator/internal/domain/externalapi"
	"github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/rendis/feature-evaluator/internal/domain/segment"
	"github.com/rendis/feature-evaluator/internal/dto"
	"github.com/rendis/feature-evaluator/internal/engine"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// RuleHandler handles rule CRUD endpoints within a feature.
type RuleHandler struct {
	featureSvc     *feature.Service
	segmentSvc     *segment.Service
	externalAPISvc *externalapi.Service
	extAPIResolver evaluation.ExternalAPIResolver
	engine         *engine.Engine
	changelogSvc   *changelog.Service
}

// NewRuleHandler creates a new RuleHandler.
func NewRuleHandler(
	featureSvc *feature.Service,
	segmentSvc *segment.Service,
	externalAPISvc *externalapi.Service,
	extAPIResolver evaluation.ExternalAPIResolver,
	eng *engine.Engine,
	changelogSvc *changelog.Service,
) *RuleHandler {
	return &RuleHandler{
		featureSvc:     featureSvc,
		segmentSvc:     segmentSvc,
		externalAPISvc: externalAPISvc,
		extAPIResolver: extAPIResolver,
		engine:         eng,
		changelogSvc:   changelogSvc,
	}
}

// List godoc
// @Summary List rules for a feature
// @Description Returns all rules for a feature, sorted by priority
// @Tags rules
// @Produce json
// @Param key path string true "Feature key"
// @Success 200 {object} dto.DataResponse[[]dto.RuleResponse]
// @Failure 404 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/features/{key}/rules [get]
func (h *RuleHandler) List(c *gin.Context) {
	key := c.Param("key")
	f, err := h.featureSvc.GetByKey(c.Request.Context(), key)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	rules := make([]dto.RuleResponse, 0, len(f.Rules))
	for i := range f.Rules {
		rules = append(rules, dto.ToRuleResponse(&f.Rules[i]))
	}

	c.JSON(http.StatusOK, gin.H{"data": rules})
}

// Create godoc
// @Summary Create a rule
// @Description Adds a new rule to a feature with expression, value, and optional rollout/bindings
// @Tags rules
// @Accept json
// @Produce json
// @Param key path string true "Feature key"
// @Param request body dto.CreateRuleRequest true "Rule creation payload"
// @Success 201 {object} dto.RuleResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/features/{key}/rules [post]
func (h *RuleHandler) Create(c *gin.Context) {
	key := c.Param("key")
	var req dto.CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	if err := h.engine.Validate(req.Expression); err != nil {
		slog.Info("invalid expression", "expressionLen", len(req.Expression), "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	if req.RolloutPercentage != nil && (*req.RolloutPercentage < 0 || *req.RolloutPercentage > 100) {
		dto.RespondError(c, apierror.NewBadRequest("rolloutPercentage must be between 0 and 100", "error.invalidRolloutPercentage"))
		return
	}
	sourceBindings, err := h.toDomainSourceBindings(c, req.SourceBindings)
	if err != nil {
		dto.RespondError(c, err)
		return
	}
	metadata, err := canonicalizeRuleMetadata(req.Metadata, req.Expression, sourceBindings)
	if err != nil {
		dto.RespondError(c, err)
		return
	}
	externalAPIBindings, err := h.toDomainExternalAPIBindings(c, req.ExternalAPIBindings)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	ruleIDValue, err := uuid.NewV7()
	if err != nil {
		dto.RespondError(c, fmt.Errorf("generating rule id: %w", err))
		return
	}

	rule := &feature.Rule{
		ID:                  ruleIDValue.String(),
		Name:                req.Name,
		Priority:            req.Priority,
		Enabled:             req.Enabled,
		Expression:          req.Expression,
		Value:               req.Value,
		RolloutPercentage:   req.RolloutPercentage,
		SourceBindings:      sourceBindings,
		ExternalAPIBindings: externalAPIBindings,
		Metadata:            metadata,
	}

	if err := h.featureSvc.AddRule(c.Request.Context(), key, rule); err != nil {
		slog.Error("creating rule", "error", err, "featureKey", key, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	recordChange(c, h.changelogSvc, &changelog.ChangeEntry{
		EntityType: changelog.EntityRule,
		EntityKey:  rule.ID,
		ParentKey:  key,
		Action:     changelog.ActionCreate,
		Metadata:   map[string]any{"ruleName": rule.Name},
	})

	c.JSON(http.StatusCreated, dto.ToRuleResponse(rule))
}

// Update godoc
// @Summary Update a rule
// @Description Updates an existing rule within a feature
// @Tags rules
// @Accept json
// @Produce json
// @Param key path string true "Feature key"
// @Param ruleId path string true "Rule ID"
// @Param request body dto.UpdateRuleRequest true "Rule update payload"
// @Success 200 {object} dto.RuleResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/features/{key}/rules/{ruleId} [put]
func (h *RuleHandler) Update(c *gin.Context) {
	key := c.Param("key")
	ruleID := c.Param("ruleId")

	var req dto.UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	if err := h.engine.Validate(req.Expression); err != nil {
		dto.RespondError(c, err)
		return
	}

	if req.RolloutPercentage != nil && (*req.RolloutPercentage < 0 || *req.RolloutPercentage > 100) {
		dto.RespondError(c, apierror.NewBadRequest("rolloutPercentage must be between 0 and 100", "error.invalidRolloutPercentage"))
		return
	}
	sourceBindings, err := h.toDomainSourceBindings(c, req.SourceBindings)
	if err != nil {
		dto.RespondError(c, err)
		return
	}
	metadata, err := canonicalizeRuleMetadata(req.Metadata, req.Expression, sourceBindings)
	if err != nil {
		dto.RespondError(c, err)
		return
	}
	externalAPIBindings, err := h.toDomainExternalAPIBindings(c, req.ExternalAPIBindings)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	rule := &feature.Rule{
		ID:                  ruleID,
		Name:                req.Name,
		Priority:            req.Priority,
		Enabled:             req.Enabled,
		Expression:          req.Expression,
		Value:               req.Value,
		RolloutPercentage:   req.RolloutPercentage,
		SourceBindings:      sourceBindings,
		ExternalAPIBindings: externalAPIBindings,
		Metadata:            metadata,
	}

	if err := h.featureSvc.UpdateRule(c.Request.Context(), key, rule); err != nil {
		slog.Error("updating rule", "error", err, "featureKey", key, "ruleId", ruleID, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	recordChange(c, h.changelogSvc, &changelog.ChangeEntry{
		EntityType: changelog.EntityRule,
		EntityKey:  ruleID,
		ParentKey:  key,
		Action:     changelog.ActionUpdate,
		Metadata:   map[string]any{"ruleName": rule.Name},
	})

	c.JSON(http.StatusOK, dto.ToRuleResponse(rule))
}

func (h *RuleHandler) toDomainExternalAPIBindings(c *gin.Context, bindings []dto.ExternalAPIBindingRequest) ([]feature.ExternalAPIBinding, error) {
	if len(bindings) == 0 {
		return nil, nil
	}
	result := make([]feature.ExternalAPIBinding, 0, len(bindings))
	for _, b := range bindings {
		if h.externalAPISvc != nil {
			api, err := h.externalAPISvc.GetByKey(c.Request.Context(), b.ExternalAPIKey)
			if err != nil {
				return nil, apierror.NewBadRequest(
					fmt.Sprintf("external API %q not found", b.ExternalAPIKey),
					"error.externalApiNotFound",
				)
			}
			if err := validateParamMappings(api, b.ParamMappings); err != nil {
				return nil, err
			}
		}
		failMode := feature.FailModeOpen
		if b.FailMode == string(feature.FailModeClosed) {
			failMode = feature.FailModeClosed
		}
		mappings := make([]feature.ParamMapping, 0, len(b.ParamMappings))
		for _, m := range b.ParamMappings {
			mappings = append(mappings, feature.ParamMapping{
				ParamName:    m.ParamName,
				Mode:         m.Mode,
				InputPath:    m.InputPath,
				LiteralValue: m.LiteralValue,
			})
		}
		result = append(result, feature.ExternalAPIBinding{
			ExternalAPIKey: b.ExternalAPIKey,
			ParamMappings:  mappings,
			FailMode:       failMode,
			CacheEnabled:   b.CacheEnabled,
			CacheTTL:       b.CacheTTL,
		})
	}
	return result, nil
}

func validateParamMappings(api *externalapi.ExternalAPI, mappings []dto.ParamMappingRequest) error {
	mappedParams := make(map[string]bool, len(mappings))
	for _, m := range mappings {
		mappedParams[m.ParamName] = true
	}
	for _, param := range api.Params {
		if param.Required && !mappedParams[param.Name] {
			return apierror.NewBadRequest(
				fmt.Sprintf("required parameter %q of external API %q is not mapped", param.Name, api.Key),
				"error.missingRequiredParam",
			)
		}
	}
	return nil
}

// Delete godoc
// @Summary Delete a rule
// @Description Removes a rule from a feature
// @Tags rules
// @Produce json
// @Param key path string true "Feature key"
// @Param ruleId path string true "Rule ID"
// @Success 200 {object} dto.MessageResponse
// @Failure 404 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/features/{key}/rules/{ruleId} [delete]
func (h *RuleHandler) Delete(c *gin.Context) {
	key := c.Param("key")
	ruleID := c.Param("ruleId")

	if err := h.featureSvc.DeleteRule(c.Request.Context(), key, ruleID); err != nil {
		dto.RespondError(c, err)
		return
	}

	recordChange(c, h.changelogSvc, &changelog.ChangeEntry{
		EntityType: changelog.EntityRule,
		EntityKey:  ruleID,
		ParentKey:  key,
		Action:     changelog.ActionDelete,
	})

	c.JSON(http.StatusOK, gin.H{"message": "rule deleted"})
}

// Reorder godoc
// @Summary Reorder rules
// @Description Sets the priority order of rules within a feature
// @Tags rules
// @Accept json
// @Produce json
// @Param key path string true "Feature key"
// @Param request body dto.ReorderRulesRequest true "Ordered list of rule IDs"
// @Success 200 {object} dto.MessageResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/features/{key}/rules/reorder [put]
func (h *RuleHandler) Reorder(c *gin.Context) {
	key := c.Param("key")
	var req dto.ReorderRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	if err := h.featureSvc.ReorderRules(c.Request.Context(), key, req.RuleIDs); err != nil {
		dto.RespondError(c, err)
		return
	}

	recordChange(c, h.changelogSvc, &changelog.ChangeEntry{
		EntityType: changelog.EntityRule,
		EntityKey:  key,
		Action:     changelog.ActionReorder,
		Metadata:   map[string]any{"ruleIds": req.RuleIDs},
	})

	c.JSON(http.StatusOK, gin.H{"message": "rules reordered"})
}

// ValidateExpression godoc
// @Summary Validate an expression
// @Description Validates an expression string without evaluating it
// @Tags expressions
// @Accept json
// @Produce json
// @Param request body dto.ValidateExpressionRequest true "Expression to validate"
// @Success 200 {object} dto.ValidateExpressionResponse
// @Failure 400 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/expression/validate [post]
func (h *RuleHandler) ValidateExpression(c *gin.Context) {
	var req dto.ValidateExpressionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	err := h.engine.Validate(req.Expression)
	if err != nil {
		errStr := err.Error()
		c.JSON(http.StatusOK, dto.ValidateExpressionResponse{
			Valid: false,
			Error: &errStr,
		})
		return
	}

	c.JSON(http.StatusOK, dto.ValidateExpressionResponse{Valid: true})
}

// TestExpression godoc
// @Summary Test an expression
// @Description Tests an expression against a provided context and returns the evaluation result
// @Tags expressions
// @Accept json
// @Produce json
// @Param request body dto.TestExpressionRequest true "Expression and context to test"
// @Success 200 {object} dto.TestExpressionResponse
// @Failure 400 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/expression/test [post]
func (h *RuleHandler) TestExpression(c *gin.Context) {
	var req dto.TestExpressionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	// Extract "authenticated" from context (it's a control field, not a namespace)
	authenticated := getBoolValue(req.Context, "authenticated")

	// Remove "authenticated" so it doesn't get injected as a namespace
	evalContext := make(map[string]any, len(req.Context))
	for k, v := range req.Context {
		if k != "authenticated" {
			evalContext[k] = v
		}
	}

	// Build a test environment with the namespaced context
	noopSegmentChecker := func(string) bool { return false }
	noopExternalAPIChecker := func(string) bool { return false }
	env := engine.BuildEnv(evalContext, engine.ExpressionInputData{}, authenticated, noopSegmentChecker, noopExternalAPIChecker)

	result, err := h.engine.CompileAndRun(req.Expression, env)
	if err != nil {
		errStr := err.Error()
		c.JSON(http.StatusOK, dto.TestExpressionResponse{
			Result:  nil,
			Matched: false,
			Error:   &errStr,
		})
		return
	}

	matched, _ := result.(bool)
	c.JSON(http.StatusOK, dto.TestExpressionResponse{
		Result:  result,
		Matched: matched,
	})
}

// FeatureExpressionSchema godoc
// @Summary Get feature expression schema
// @Description Returns the expression inputs available for a specific feature based on its input contract
// @Tags expressions
// @Produce json
// @Param key path string true "Feature key"
// @Success 200 {object} dto.FeatureExpressionSchemaResponse
// @Failure 404 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/features/{key}/expression-schema [get]
func (h *RuleHandler) FeatureExpressionSchema(c *gin.Context) {
	featureKey := c.Param("key")
	f, err := h.featureSvc.GetByKey(c.Request.Context(), featureKey)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FeatureExpressionSchemaResponse{
		Headers:      buildHeaderExpressionFields(f.InputContract.Headers),
		RequestBody:  buildRequestBodyExpressionFields(f.InputContract.RequestBodySchema, f.InputContract.RequestBodyExample),
		Derived:      buildDerivedExpressionFields(),
		AdvancedMode: true,
	})
}

// FeatureTestExpression godoc
// @Summary Test expression with feature context
// @Description Tests an expression using the feature-specific input contract with a simulated scenario
// @Tags expressions
// @Accept json
// @Produce json
// @Param key path string true "Feature key"
// @Param request body dto.FeatureExpressionTestRequest true "Expression, scenario, and bindings to test"
// @Success 200 {object} dto.FeatureExpressionTestResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/features/{key}/expression/test [post]
func (h *RuleHandler) FeatureTestExpression(c *gin.Context) { //nolint:funlen // multi-step expression test
	featureKey := c.Param("key")
	f, err := h.featureSvc.GetByKey(c.Request.Context(), featureKey)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	var req dto.FeatureExpressionTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}
	if err := h.engine.Validate(req.Expression); err != nil {
		errStr := err.Error()
		c.JSON(http.StatusOK, dto.FeatureExpressionTestResponse{
			Result:  nil,
			Matched: false,
			Error:   &errStr,
		})
		return
	}

	sourceBindings, err := h.toDomainSourceBindings(c, req.SourceBindings)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	rawInput := buildFeatureTestInput(req.Scenario)
	authInput, _ := rawInput["auth"].(map[string]any)
	authState := evaluation.AuthValidationResult{
		Authenticated: strings.TrimSpace(fmt.Sprintf("%v", authInput["bearerToken"])) != "",
	}
	evalContext := map[string]any{}
	if legacyContext, ok := req.Scenario.RequestBody["context"].(map[string]any); ok {
		evalContext = legacyContext
	}

	prepared := evaluation.PrepareExpressionInput(f.InputContract, rawInput, authState, evalContext)
	resolvedSources, err := evaluation.ResolveSegmentSources(c.Request.Context(), h.segmentSvc, sourceBindings, &prepared)
	if err != nil {
		errStr := err.Error()
		c.JSON(http.StatusOK, dto.FeatureExpressionTestResponse{
			Result:  nil,
			Matched: false,
			Derived: prepared.Derived,
			Error:   &errStr,
		})
		return
	}

	noopSegmentChecker := func(string) bool { return false }
	noopExtAPIChecker := func(string) bool { return false }
	env := engine.BuildEnv(
		evalContext,
		engine.ExpressionInputData{
			Headers:     prepared.Headers,
			RequestBody: prepared.RequestBody,
			Derived:     prepared.Derived,
			Sources:     prepared.Sources,
		},
		authState.Authenticated,
		noopSegmentChecker,
		noopExtAPIChecker,
	)

	// Resolve external API bindings if present
	if len(req.ExternalAPIBindings) > 0 && h.extAPIResolver != nil {
		domainBindings, bindErr := h.toDomainExternalAPIBindings(c, req.ExternalAPIBindings)
		if bindErr == nil {
			extResults := h.extAPIResolver.Resolve(c.Request.Context(), domainBindings, env)
			env["externalApi"] = engine.ExternalAPIChecker(func(apiKey string) bool {
				return extResults[apiKey]
			})
		}
	}

	result, err := h.engine.CompileAndRun(req.Expression, env)
	if err != nil {
		errStr := err.Error()
		c.JSON(http.StatusOK, dto.FeatureExpressionTestResponse{
			Result:          nil,
			Matched:         false,
			Derived:         prepared.Derived,
			ResolvedSources: toResolvedSourceResponses(resolvedSources),
			Error:           &errStr,
		})
		return
	}

	matched, _ := result.(bool)
	c.JSON(http.StatusOK, dto.FeatureExpressionTestResponse{
		Result:          result,
		Matched:         matched,
		Derived:         prepared.Derived,
		ResolvedSources: toResolvedSourceResponses(resolvedSources),
		Explanation:     featureExpressionExplanation(matched, resolvedSources),
	})
}

// ExpressionSchema godoc
// @Summary Get expression schema
// @Description Returns the available fields, functions, and operators for building expressions
// @Tags expressions
// @Produce json
// @Success 200 {object} dto.ExpressionSchemaResponse
// @Security BearerAuth
// @Router /admin/expression/schema [get]
func (h *RuleHandler) ExpressionSchema(c *gin.Context) {
	schema := gin.H{
		"fields": []gin.H{
			{"name": "user", "type": "object", "description": "User attributes (id, email, plan, etc.). Access: user.id, user.email, user.plan"},
			{"name": "tenant", "type": "object", "description": "Tenant context (id, plan, country, etc.). Access: tenant.id, tenant.plan"},
			{"name": "campus", "type": "object", "description": "Campus context (id, region, etc.). Access: campus.id, campus.region"},
			{"name": "program", "type": "object", "description": "Program context (id, etc.). Access: program.id"},
			{"name": "authenticated", "type": "boolean", "description": "Whether the request is authenticated"},
		},
		"functions": []gin.H{
			{"name": "inSegment", "signature": "inSegment(segmentKey string) bool", "description": "Check if user belongs to a segment"},
			{"name": "now", "signature": "now() time.Time", "description": "Current UTC time"},
			{"name": "dateBefore", "signature": "dateBefore(date, reference string) bool", "description": "Check if date is before reference"},
			{"name": "dateAfter", "signature": "dateAfter(date, reference string) bool", "description": "Check if date is after reference"},
		},
		"operators": []string{"==", "!=", ">", ">=", "<", "<=", "&&", "||", "!", "in", "not in", "contains", "startsWith", "endsWith", "matches"},
		"notes":     "Any namespace can be added to the context map. Standard namespaces are: user, tenant, campus, program. Each namespace becomes a top-level variable in expressions.",
	}

	c.JSON(http.StatusOK, schema)
}

func getBoolValue(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func (h *RuleHandler) toDomainSourceBindings(c *gin.Context, req dto.SourceBindingsRequest) (feature.SourceBindings, error) {
	segments := make([]feature.SegmentSourceBinding, 0, len(req.Segments))
	for _, binding := range req.Segments {
		if h.segmentSvc != nil {
			if _, err := h.segmentSvc.GetByKey(c.Request.Context(), binding.SegmentKey); err != nil {
				return feature.SourceBindings{}, err
			}
		}
		segments = append(segments, feature.SegmentSourceBinding{
			SegmentKey: binding.SegmentKey,
			LookupPath: binding.LookupPath,
		})
	}
	return feature.SourceBindings{Segments: segments}, nil
}

func buildHeaderExpressionFields(headers []feature.InputHeader) []dto.FeatureExpressionFieldResponse {
	fields := make([]dto.FeatureExpressionFieldResponse, 0, len(headers))
	for _, header := range headers {
		label := header.Label
		if label == "" {
			label = header.HeaderName
		}
		fields = append(fields, dto.FeatureExpressionFieldResponse{
			Path:        "headers." + header.ExpressionKey,
			Label:       label,
			Description: header.Description,
			Type:        string(header.Type),
			Example:     header.HeaderName,
			Group:       "headers",
		})
	}
	return fields
}

func buildRequestBodyExpressionFields(schema map[string]any, example map[string]any) []dto.FeatureExpressionFieldResponse {
	fields := make([]dto.FeatureExpressionFieldResponse, 0)
	appendSchemaFields(&fields, schema, example, "", "requestBody")
	return fields
}

func appendSchemaFields(
	fields *[]dto.FeatureExpressionFieldResponse,
	schema map[string]any,
	example map[string]any,
	prefix string,
	group string,
) {
	properties, _ := schema["properties"].(map[string]any)
	if len(properties) == 0 {
		return
	}
	for key, rawChild := range properties {
		child, ok := rawChild.(map[string]any)
		if !ok {
			continue
		}
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		childType, _ := child["type"].(string)
		if childType == "object" {
			childExample, _ := example[key].(map[string]any)
			appendSchemaFields(fields, child, childExample, path, group)
			continue
		}
		*fields = append(*fields, dto.FeatureExpressionFieldResponse{
			Path:    "requestBody." + path,
			Label:   path,
			Type:    childType,
			Example: example[key],
			Group:   group,
		})
	}
}

func buildDerivedExpressionFields() []dto.FeatureExpressionFieldResponse {
	return []dto.FeatureExpressionFieldResponse{
		{Path: "derived.authenticated", Label: "Autenticado", Type: "boolean", Group: "derived"},
		{Path: "derived.bearerTokenPresent", Label: "Bearer presente", Type: "boolean", Group: "derived"},
		{Path: "derived.apiKeyPresent", Label: "API key presente", Type: "boolean", Group: "derived"},
		{Path: "derived.userId", Label: "User ID", Type: "string", Group: "derived"},
		{Path: "derived.subject", Label: "Subject", Type: "string", Group: "derived"},
		{Path: "derived.email", Label: "Email", Type: "string", Group: "derived"},
		{Path: "derived.name", Label: "Nombre", Type: "string", Group: "derived"},
	}
}

func buildFeatureTestInput(scenario dto.FeatureTestScenario) map[string]any {
	headers := make(map[string]any, len(scenario.Headers))
	for key, value := range scenario.Headers {
		headers[strings.ToLower(key)] = value
	}
	bearerToken := ""
	if rawAuth, ok := scenario.Headers["Authorization"]; ok {
		if strings.HasPrefix(strings.TrimSpace(rawAuth), "Bearer ") {
			bearerToken = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(rawAuth), "Bearer "))
		}
	}
	if rawAuth, ok := scenario.Headers["authorization"]; ok && bearerToken == "" {
		if strings.HasPrefix(strings.TrimSpace(rawAuth), "Bearer ") {
			bearerToken = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(rawAuth), "Bearer "))
		}
	}
	apiKey := scenario.Headers["X-Api-Key"]
	if apiKey == "" {
		apiKey = scenario.Headers["x-api-key"]
	}

	return map[string]any{
		"headers": headers,
		"body":    scenario.RequestBody,
		"auth": map[string]any{
			"bearerToken": bearerToken,
			"apiKey":      apiKey,
		},
	}
}

func toResolvedSourceResponses(sources []evaluation.ResolvedSegmentSource) []dto.ResolvedSegmentSourceResponse {
	result := make([]dto.ResolvedSegmentSourceResponse, 0, len(sources))
	for _, source := range sources {
		result = append(result, dto.ResolvedSegmentSourceResponse{
			SegmentKey:  source.SegmentKey,
			LookupPath:  source.LookupPath,
			LookupValue: source.LookupValue,
			Found:       source.Found,
			Data:        source.Data,
		})
	}
	return result
}

func featureExpressionExplanation(
	matched bool,
	sources []evaluation.ResolvedSegmentSource,
) string {
	if matched {
		for _, source := range sources {
			if source.Found {
				return fmt.Sprintf("La regla matcheo y resolvio datos desde %s usando %s.", source.SegmentKey, source.LookupPath)
			}
		}
		return "La regla matcheo con los inputs simulados."
	}
	for _, source := range sources {
		if !source.Found {
			return fmt.Sprintf("No hubo match; no se encontro un registro en %s para %s.", source.SegmentKey, source.LookupPath)
		}
	}
	return "La regla no hizo match con los inputs simulados."
}
