package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/rendis/feature-evaluator/internal/domain/changelog"
	"github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/rendis/feature-evaluator/internal/domain/pack"
	"github.com/rendis/feature-evaluator/internal/domain/tag"
	"github.com/rendis/feature-evaluator/internal/domain/tier"
	"github.com/rendis/feature-evaluator/internal/dto"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

const maxTagsPerFeature = 10

// FeatureHandler handles feature CRUD endpoints.
type FeatureHandler struct {
	svc          *feature.Service
	tagSvc       *tag.Service
	packSvc      *pack.Service
	changelogSvc *changelog.Service
}

// NewFeatureHandler creates a new FeatureHandler.
func NewFeatureHandler(svc *feature.Service, tagSvc *tag.Service, packSvc *pack.Service, changelogSvc *changelog.Service) *FeatureHandler {
	return &FeatureHandler{svc: svc, tagSvc: tagSvc, packSvc: packSvc, changelogSvc: changelogSvc}
}

// buildTagMap collects all unique tag keys from features, looks them up, and returns a map.
func (h *FeatureHandler) buildTagMap(c *gin.Context, features []feature.Feature) map[string]tag.Tag {
	keySet := make(map[string]struct{})
	for i := range features {
		for _, k := range features[i].Tags {
			keySet[k] = struct{}{}
		}
	}
	if len(keySet) == 0 {
		return nil
	}
	keys := make([]string, 0, len(keySet))
	for k := range keySet {
		keys = append(keys, k)
	}
	tags, err := h.tagSvc.FindByKeys(c.Request.Context(), keys)
	if err != nil {
		slog.Warn("fetching tags for enrichment", "error", err)
		return nil
	}
	m := make(map[string]tag.Tag, len(tags))
	for i := range tags {
		m[tags[i].Key] = tags[i]
	}
	return m
}

// buildTagMapForFeature builds a tag map for a single feature.
func (h *FeatureHandler) buildTagMapForFeature(c *gin.Context, f *feature.Feature) map[string]tag.Tag {
	return h.buildTagMap(c, []feature.Feature{*f})
}

// buildPackRefs returns pack refs for a single feature.
func (h *FeatureHandler) buildPackRefs(c *gin.Context, featureKey string) []dto.PackRef {
	if h.packSvc == nil {
		return nil
	}
	packs, err := h.packSvc.FindByFeatureKey(c.Request.Context(), featureKey)
	if err != nil {
		//nolint:gosec // Structured logging records the feature key for operator debugging only.
		slog.Warn("fetching packs for feature", "featureKey", featureKey, "error", err, "requestId", middleware.GetRequestID(c))
		return nil
	}
	return dto.PacksToRefs(packs)
}

// buildPackMap returns a map of featureKey -> []PackRef for all packs.
func (h *FeatureHandler) buildPackMap(c *gin.Context) map[string][]dto.PackRef {
	if h.packSvc == nil {
		return nil
	}
	allPacks, err := h.packSvc.List(c.Request.Context())
	if err != nil {
		slog.Warn("fetching packs for feature list", "error", err, "requestId", middleware.GetRequestID(c))
		return nil
	}
	m := make(map[string][]dto.PackRef)
	for i := range allPacks {
		ref := dto.PackRef{Key: allPacks[i].Key, Name: allPacks[i].Name}
		for _, fk := range allPacks[i].FeatureKeys {
			m[fk] = append(m[fk], ref)
		}
	}
	return m
}

// buildTierRefs resolves tier refs for a single feature via its packs.
func (h *FeatureHandler) buildTierRefs(c *gin.Context, featureKey string) []dto.TierRef {
	if h.packSvc == nil {
		return nil
	}
	tierKeys := h.packSvc.ResolveTierKeysForFeature(c.Request.Context(), featureKey)
	if len(tierKeys) == 0 {
		return []dto.TierRef{}
	}
	tiers := tier.FindByKeys(tierKeys)
	return dto.TiersToRefs(tiers)
}

// collectFeatureTierKeys scans packs and returns a mapping from featureKey to
// its set of tierKeys, plus the deduplicated set of all tierKeys found.
func collectFeatureTierKeys(packs []pack.Pack) (featureTierKeys map[string]map[string]struct{}, allTierKeys map[string]struct{}) {
	featureTierKeys = make(map[string]map[string]struct{})
	allTierKeys = make(map[string]struct{})
	for i := range packs {
		if packs[i].TierKey == nil || *packs[i].TierKey == "" {
			continue
		}
		tk := *packs[i].TierKey
		allTierKeys[tk] = struct{}{}
		for _, fk := range packs[i].FeatureKeys {
			if featureTierKeys[fk] == nil {
				featureTierKeys[fk] = make(map[string]struct{})
			}
			featureTierKeys[fk][tk] = struct{}{}
		}
	}
	return featureTierKeys, allTierKeys
}

// buildTierMap returns a map of featureKey -> []TierRef for all features.
func (h *FeatureHandler) buildTierMap(c *gin.Context, _ []feature.Feature) map[string][]dto.TierRef {
	if h.packSvc == nil {
		return nil
	}
	allPacks, err := h.packSvc.List(c.Request.Context())
	if err != nil {
		//nolint:gosec // Structured logging, no user-controlled data in tier map enrichment path.
		slog.Warn("fetching packs for tier map", "error", err, "requestId", middleware.GetRequestID(c))
		return nil
	}
	featureTierKeys, allTierKeys := collectFeatureTierKeys(allPacks)
	if len(allTierKeys) == 0 {
		return nil
	}
	keys := make([]string, 0, len(allTierKeys))
	for k := range allTierKeys {
		keys = append(keys, k)
	}
	tiers := tier.FindByKeys(keys)
	tierByKeyMap := make(map[string]dto.TierRef, len(tiers))
	for i := range tiers {
		tierByKeyMap[tiers[i].Key] = dto.ToTierRef(&tiers[i])
	}
	result := make(map[string][]dto.TierRef)
	for fk, tkSet := range featureTierKeys {
		refs := make([]dto.TierRef, 0, len(tkSet))
		for tk := range tkSet {
			if ref, ok := tierByKeyMap[tk]; ok {
				refs = append(refs, ref)
			}
		}
		result[fk] = refs
	}
	return result
}

func applyOptionalFeatureFields(
	existing *feature.Feature,
	req dto.UpdateFeatureRequest,
	payload map[string]json.RawMessage,
	activeFrom *time.Time,
	activeUntil *time.Time,
	trialUntil *time.Time,
) {
	applyOptionalPointerField(payload, "enabled", req.Enabled, func(value bool) {
		existing.Enabled = value
	})
	applyOptionalPointerField(payload, "evalCacheEnabled", req.EvalCacheEnabled, func(value bool) {
		existing.EvalCacheEnabled = value
	})
	applyOptionalPointerField(payload, "evalCacheTTLSeconds", req.EvalCacheTTLSeconds, func(value int) {
		existing.EvalCacheTTLSeconds = value
	})
	applyOptionalValueField(payload, "valueType", func() {
		existing.ValueType = feature.ValueType(req.ValueType)
	})
	applyOptionalValueField(payload, "defaultValue", func() {
		existing.DefaultValue = req.DefaultValue
	})
	applyOptionalValueField(payload, "activeFrom", func() {
		existing.ActiveFrom = activeFrom
	})
	applyOptionalValueField(payload, "activeUntil", func() {
		existing.ActiveUntil = activeUntil
	})
	applyOptionalValueField(payload, "environments", func() {
		existing.Environments = req.Environments
	})
	applyOptionalValueField(payload, "accessPolicy", func() {
		existing.AccessPolicy = feature.AccessPolicy(req.AccessPolicy)
	})
	applyOptionalValueField(payload, "authProfileKey", func() {
		existing.AuthProfileKey = req.AuthProfileKey
	})
	applyOptionalValueField(payload, "inputContract", func() {
		existing.InputContract = toDomainInputContract(req.InputContract)
	})
	applyOptionalValueField(payload, "metadata", func() {
		existing.Metadata = req.Metadata
	})
	applyOptionalValueField(payload, "tags", func() {
		existing.Tags = req.Tags
	})
	applyOptionalValueField(payload, "trialUntil", func() {
		existing.TrialUntil = trialUntil
	})
	applyOptionalValueField(payload, "trialValue", func() {
		existing.TrialValue = req.TrialValue
	})
}

func applyOptionalValueField(payload map[string]json.RawMessage, key string, apply func()) {
	if _, ok := payload[key]; !ok {
		return
	}
	apply()
}

func applyOptionalPointerField[T any](payload map[string]json.RawMessage, key string, value *T, apply func(T)) {
	if value == nil {
		return
	}
	applyOptionalValueField(payload, key, func() {
		apply(*value)
	})
}

func toDomainInputContract(req dto.InputContractRequest) feature.InputContract {
	headers := make([]feature.InputHeader, 0, len(req.Headers))
	for _, header := range req.Headers {
		headers = append(headers, feature.InputHeader{
			HeaderName:    header.HeaderName,
			ExpressionKey: header.ExpressionKey,
			Label:         header.Label,
			Type:          feature.InputValueType(header.Type),
			Required:      header.Required,
			Description:   header.Description,
		})
	}
	return feature.InputContract{
		Headers:            headers,
		RequestBodyExample: req.RequestBodyExample,
	}
}

// List godoc
// @Summary List features
// @Description Returns a paginated list of features with optional filters
// @Tags features
// @Produce json
// @Param search query string false "Filter by name or key"
// @Param sort query string false "Sort field"
// @Param order query string false "Sort order" default(desc) Enums(asc, desc)
// @Param view query string false "Response view mode" Enums(summary)
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Items per page" default(20)
// @Param enabled query boolean false "Filter by enabled status"
// @Param valueType query string false "Filter by value type"
// @Param tags query []string false "Filter by tag keys" collectionFormat(multi)
// @Param environment query string false "Filter by environment"
// @Success 200 {object} dto.ListResponse[dto.FeatureSummaryResponse]
// @Failure 500 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/features [get]
func (h *FeatureHandler) List(c *gin.Context) {
	params := feature.ListParams{
		Search:    c.Query("search"),
		SortBy:    c.Query("sort"),
		SortOrder: c.DefaultQuery("order", "desc"),
	}
	if c.Query("view") == string(feature.ListViewSummary) {
		params.View = feature.ListViewSummary
	}

	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil {
		params.Page = page
	}
	if ps, err := strconv.Atoi(c.DefaultQuery("pageSize", "20")); err == nil {
		params.PageSize = ps
	}
	if enabled := c.Query("enabled"); enabled != "" {
		b := enabled == "true"
		params.Enabled = &b
	}
	if vt := c.Query("valueType"); vt != "" {
		vtype := feature.ValueType(vt)
		params.ValueType = &vtype
	}
	if tags := c.QueryArray("tags"); len(tags) > 0 {
		params.Tags = tags
	}
	if env := c.Query("environment"); env != "" {
		params.Environment = env
	}

	result, err := h.svc.List(c.Request.Context(), params)
	if err != nil {
		slog.Error("listing features", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	tagMap := h.buildTagMap(c, result.Data)
	tierMap := h.buildTierMap(c, result.Data)
	if params.View == feature.ListViewSummary {
		data := make([]dto.FeatureSummaryResponse, 0, len(result.Data))
		for i := range result.Data {
			data = append(data, dto.ToFeatureSummaryResponse(&result.Data[i], tagMap, tierMap[result.Data[i].Key]))
		}

		c.JSON(http.StatusOK, dto.ListResponse[dto.FeatureSummaryResponse]{
			Data: data,
			Pagination: dto.PaginationResponse{
				Page:       result.Page,
				PageSize:   result.PageSize,
				Total:      result.Total,
				TotalPages: result.TotalPages,
			},
		})
		return
	}

	packMap := h.buildPackMap(c)

	data := make([]dto.FeatureResponse, 0, len(result.Data))
	for i := range result.Data {
		data = append(data, dto.ToFeatureResponse(&result.Data[i], tagMap, tierMap[result.Data[i].Key], packMap[result.Data[i].Key]))
	}

	c.JSON(http.StatusOK, dto.ListResponse[dto.FeatureResponse]{
		Data: data,
		Pagination: dto.PaginationResponse{
			Page:       result.Page,
			PageSize:   result.PageSize,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}

// Get godoc
// @Summary Get feature detail
// @Description Returns a single feature with its rules, tags, packs, and tier references
// @Tags features
// @Produce json
// @Param key path string true "Feature key"
// @Success 200 {object} dto.FeatureDetailResponse
// @Failure 404 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/features/{key} [get]
func (h *FeatureHandler) Get(c *gin.Context) {
	key := c.Param("key")
	f, err := h.svc.GetByKey(c.Request.Context(), key)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	tagMap := h.buildTagMapForFeature(c, f)
	tierRefs := h.buildTierRefs(c, f.Key)
	packRefs := h.buildPackRefs(c, f.Key)
	c.JSON(http.StatusOK, dto.ToFeatureDetailResponse(f, tagMap, tierRefs, packRefs))
}

// Create godoc
// @Summary Create a feature
// @Description Creates a new feature flag with the given configuration
// @Tags features
// @Accept json
// @Produce json
// @Param request body dto.CreateFeatureRequest true "Feature creation payload"
// @Success 201 {object} dto.FeatureDetailResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/features [post]
func (h *FeatureHandler) Create(c *gin.Context) {
	var req dto.CreateFeatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	activeFrom, err := dto.ParseTimePtr(req.ActiveFrom)
	if err != nil {
		dto.RespondError(c, apierror.NewBadRequest(err.Error(), "error.invalidActiveFrom"))
		return
	}
	activeUntil, err := dto.ParseTimePtr(req.ActiveUntil)
	if err != nil {
		dto.RespondError(c, apierror.NewBadRequest(err.Error(), "error.invalidActiveUntil"))
		return
	}

	if len(req.Tags) > maxTagsPerFeature {
		dto.RespondError(c, apierror.NewBadRequest("a feature can have at most 10 tags", "error.tooManyTags"))
		return
	}

	trialUntil, err := dto.ParseTimePtr(req.TrialUntil)
	if err != nil {
		dto.RespondError(c, apierror.NewBadRequest(err.Error(), "error.invalidTrialUntil"))
		return
	}
	if trialUntil != nil && req.TrialValue == nil {
		dto.RespondError(c, apierror.NewBadRequest("trialValue is required when trialUntil is set", "error.trialValueRequired"))
		return
	}

	f := &feature.Feature{
		Key:                 req.Key,
		Name:                req.Name,
		Description:         req.Description,
		Enabled:             req.Enabled,
		EvalCacheEnabled:    req.EvalCacheEnabled,
		EvalCacheTTLSeconds: req.EvalCacheTTLSeconds,
		ValueType:           feature.ValueType(req.ValueType),
		DefaultValue:        req.DefaultValue,
		ActiveFrom:          activeFrom,
		ActiveUntil:         activeUntil,
		Environments:        req.Environments,
		AccessPolicy:        feature.AccessPolicy(req.AccessPolicy),
		AuthProfileKey:      req.AuthProfileKey,
		InputContract:       toDomainInputContract(req.InputContract),
		Metadata:            req.Metadata,
		Tags:                req.Tags,
		TrialUntil:          trialUntil,
		TrialValue:          req.TrialValue,
		CreatedBy:           middleware.GetUserEmail(c),
		UpdatedBy:           middleware.GetUserEmail(c),
	}

	if err := h.svc.Create(c.Request.Context(), f); err != nil {
		slog.Error("creating feature", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	recordChange(c, h.changelogSvc, &changelog.ChangeEntry{
		EntityType: changelog.EntityFeature,
		EntityKey:  f.Key,
		Action:     changelog.ActionCreate,
	})

	tagMap := h.buildTagMapForFeature(c, f)
	tierRefs := h.buildTierRefs(c, f.Key)
	packRefs := h.buildPackRefs(c, f.Key)
	c.JSON(http.StatusCreated, dto.ToFeatureDetailResponse(f, tagMap, tierRefs, packRefs))
}

// Update godoc
// @Summary Update a feature
// @Description Updates an existing feature flag (partial update supported)
// @Tags features
// @Accept json
// @Produce json
// @Param key path string true "Feature key"
// @Param request body dto.UpdateFeatureRequest true "Feature update payload"
// @Success 200 {object} dto.FeatureDetailResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/features/{key} [put]
func (h *FeatureHandler) Update(c *gin.Context) {
	key := c.Param("key")
	var req dto.UpdateFeatureRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		dto.RespondError(c, err)
		return
	}

	var payload map[string]json.RawMessage
	if err := c.ShouldBindBodyWith(&payload, binding.JSON); err != nil {
		dto.RespondError(c, err)
		return
	}

	existing, err := h.svc.GetByKey(c.Request.Context(), key)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	activeFrom, err := dto.ParseTimePtr(req.ActiveFrom)
	if err != nil {
		dto.RespondError(c, apierror.NewBadRequest(err.Error(), "error.invalidActiveFrom"))
		return
	}
	activeUntil, err := dto.ParseTimePtr(req.ActiveUntil)
	if err != nil {
		dto.RespondError(c, apierror.NewBadRequest(err.Error(), "error.invalidActiveUntil"))
		return
	}

	if len(req.Tags) > maxTagsPerFeature {
		dto.RespondError(c, apierror.NewBadRequest("a feature can have at most 10 tags", "error.tooManyTags"))
		return
	}

	trialUntil, err := dto.ParseTimePtr(req.TrialUntil)
	if err != nil {
		dto.RespondError(c, apierror.NewBadRequest(err.Error(), "error.invalidTrialUntil"))
		return
	}
	if trialUntil != nil && req.TrialValue == nil {
		dto.RespondError(c, apierror.NewBadRequest("trialValue is required when trialUntil is set", "error.trialValueRequired"))
		return
	}

	// Snapshot for diff before mutation.
	oldSnapshot := *existing

	existing.Name = req.Name
	existing.Description = req.Description
	applyOptionalFeatureFields(existing, req, payload, activeFrom, activeUntil, trialUntil)
	existing.UpdatedBy = middleware.GetUserEmail(c)

	if err := h.svc.Update(c.Request.Context(), existing); err != nil {
		slog.Error("updating feature", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	recordChange(c, h.changelogSvc, &changelog.ChangeEntry{
		EntityType:   changelog.EntityFeature,
		EntityKey:    existing.Key,
		Action:       changelog.ActionUpdate,
		FieldChanges: changelog.ComputeDiff(oldSnapshot, *existing),
	})

	tagMap := h.buildTagMapForFeature(c, existing)
	tierRefs := h.buildTierRefs(c, existing.Key)
	packRefs := h.buildPackRefs(c, existing.Key)
	c.JSON(http.StatusOK, dto.ToFeatureDetailResponse(existing, tagMap, tierRefs, packRefs))
}

// Delete godoc
// @Summary Delete a feature
// @Description Permanently removes a feature flag and all its rules
// @Tags features
// @Produce json
// @Param key path string true "Feature key"
// @Success 200 {object} dto.MessageResponse
// @Failure 404 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/features/{key} [delete]
func (h *FeatureHandler) Delete(c *gin.Context) {
	key := c.Param("key")
	if err := h.svc.Delete(c.Request.Context(), key); err != nil {
		dto.RespondError(c, err)
		return
	}

	recordChange(c, h.changelogSvc, &changelog.ChangeEntry{
		EntityType: changelog.EntityFeature,
		EntityKey:  key,
		Action:     changelog.ActionDelete,
	})

	c.JSON(http.StatusOK, gin.H{"message": "feature deleted"})
}

// ListEnvironments godoc
// @Summary List environments
// @Description Returns the static list of valid environments
// @Tags features
// @Produce json
// @Success 200 {array} string
// @Security BearerAuth
// @Router /admin/environments [get]
func (h *FeatureHandler) ListEnvironments(c *gin.Context) {
	c.JSON(http.StatusOK, feature.AllEnvironments())
}

// Toggle godoc
// @Summary Toggle a feature
// @Description Enables or disables a feature flag
// @Tags features
// @Accept json
// @Produce json
// @Param key path string true "Feature key"
// @Param request body dto.ToggleFeatureRequest true "Toggle payload with enabled flag"
// @Success 200 {object} dto.ToggleMessageResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/features/{key}/toggle [patch]
func (h *FeatureHandler) Toggle(c *gin.Context) {
	key := c.Param("key")
	var req dto.ToggleFeatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	if err := h.svc.Toggle(c.Request.Context(), key, req.Enabled, middleware.GetUserEmail(c)); err != nil {
		dto.RespondError(c, err)
		return
	}

	recordChange(c, h.changelogSvc, &changelog.ChangeEntry{
		EntityType: changelog.EntityFeature,
		EntityKey:  key,
		Action:     changelog.ActionToggle,
		FieldChanges: []changelog.FieldChange{
			{Field: "enabled", NewValue: req.Enabled},
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "feature toggled", "enabled": req.Enabled})
}
