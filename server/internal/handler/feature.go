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

func applyOptionalFeatureFields(
	existing *feature.Feature,
	req dto.UpdateFeatureRequest,
	payload map[string]json.RawMessage,
	activeFrom *time.Time,
	activeUntil *time.Time,
) {
	if _, ok := payload["enabled"]; ok && req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if _, ok := payload["valueType"]; ok {
		existing.ValueType = feature.ValueType(req.ValueType)
	}
	if _, ok := payload["defaultValue"]; ok {
		existing.DefaultValue = req.DefaultValue
	}
	if _, ok := payload["activeFrom"]; ok {
		existing.ActiveFrom = activeFrom
	}
	if _, ok := payload["activeUntil"]; ok {
		existing.ActiveUntil = activeUntil
	}
	if _, ok := payload["environments"]; ok {
		existing.Environments = req.Environments
	}
	if _, ok := payload["accessPolicy"]; ok {
		existing.AccessPolicy = feature.AccessPolicy(req.AccessPolicy)
	}
	if _, ok := payload["authProfileKey"]; ok {
		existing.AuthProfileKey = req.AuthProfileKey
	}
	if _, ok := payload["inputContract"]; ok {
		existing.InputContract = toDomainInputContract(req.InputContract)
	}
	if _, ok := payload["metadata"]; ok {
		existing.Metadata = req.Metadata
	}
	if _, ok := payload["tags"]; ok {
		existing.Tags = req.Tags
	}
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

// List returns a paginated list of features.
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
	if params.View == feature.ListViewSummary {
		data := make([]dto.FeatureSummaryResponse, 0, len(result.Data))
		for i := range result.Data {
			data = append(data, dto.ToFeatureSummaryResponse(&result.Data[i], tagMap))
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
		data = append(data, dto.ToFeatureResponse(&result.Data[i], tagMap, packMap[result.Data[i].Key]))
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

// Get returns a single feature with its rules.
func (h *FeatureHandler) Get(c *gin.Context) {
	key := c.Param("key")
	f, err := h.svc.GetByKey(c.Request.Context(), key)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	tagMap := h.buildTagMapForFeature(c, f)
	packRefs := h.buildPackRefs(c, f.Key)
	c.JSON(http.StatusOK, dto.ToFeatureDetailResponse(f, tagMap, packRefs))
}

// Create creates a new feature.
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

	f := &feature.Feature{
		Key:            req.Key,
		Name:           req.Name,
		Description:    req.Description,
		Enabled:        req.Enabled,
		ValueType:      feature.ValueType(req.ValueType),
		DefaultValue:   req.DefaultValue,
		ActiveFrom:     activeFrom,
		ActiveUntil:    activeUntil,
		Environments:   req.Environments,
		AccessPolicy:   feature.AccessPolicy(req.AccessPolicy),
		AuthProfileKey: req.AuthProfileKey,
		InputContract:  toDomainInputContract(req.InputContract),
		Metadata:       req.Metadata,
		Tags:           req.Tags,
		CreatedBy:      middleware.GetUserEmail(c),
		UpdatedBy:      middleware.GetUserEmail(c),
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
	packRefs := h.buildPackRefs(c, f.Key)
	c.JSON(http.StatusCreated, dto.ToFeatureDetailResponse(f, tagMap, packRefs))
}

// Update updates an existing feature.
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

	// Snapshot for diff before mutation.
	oldSnapshot := *existing

	existing.Name = req.Name
	existing.Description = req.Description
	applyOptionalFeatureFields(existing, req, payload, activeFrom, activeUntil)
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
	packRefs := h.buildPackRefs(c, existing.Key)
	c.JSON(http.StatusOK, dto.ToFeatureDetailResponse(existing, tagMap, packRefs))
}

// Delete removes a feature.
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

// ListEnvironments returns the static list of valid environments.
func (h *FeatureHandler) ListEnvironments(c *gin.Context) {
	c.JSON(http.StatusOK, feature.AllEnvironments())
}

// Toggle enables or disables a feature.
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
