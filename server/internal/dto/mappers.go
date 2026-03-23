package dto

import (
	"fmt"
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/audit"
	"github.com/rendis/feature-evaluator/internal/domain/authprofile"
	"github.com/rendis/feature-evaluator/internal/domain/externalapi"
	"github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/rendis/feature-evaluator/internal/domain/member"
	"github.com/rendis/feature-evaluator/internal/domain/pack"
	"github.com/rendis/feature-evaluator/internal/domain/segment"
	"github.com/rendis/feature-evaluator/internal/domain/tag"
	"github.com/rendis/feature-evaluator/internal/domain/tier"
	"github.com/rendis/feature-evaluator/internal/engine"
)

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := formatTime(*t)
	return &s
}

// ParseTimePtr parses an ISO 8601 string pointer to *time.Time.
// Returns nil if the input is nil or empty.
func ParseTimePtr(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil, fmt.Errorf("invalid time format %q: expected RFC3339 (e.g. 2026-03-01T00:00:00Z)", *s)
	}
	utc := t.UTC()
	return &utc, nil
}

// ToMemberResponse maps a domain member to its response DTO.
func ToMemberResponse(m *member.Member) MemberResponse {
	return MemberResponse{
		ID:          m.ID,
		Email:       m.Email,
		Role:        string(m.Role),
		DisplayName: m.DisplayName,
		AddedBy:     m.AddedBy,
		CreatedAt:   formatTime(m.CreatedAt),
		UpdatedAt:   formatTime(m.UpdatedAt),
	}
}

// ToTagResponse maps a domain tag to its response DTO.
func ToTagResponse(t *tag.Tag) TagDetailResponse {
	return TagDetailResponse{
		ID:        t.ID,
		Key:       t.Key,
		Name:      t.Name,
		Color:     t.Color,
		CreatedAt: formatTime(t.CreatedAt),
		UpdatedAt: formatTime(t.UpdatedAt),
		CreatedBy: t.CreatedBy,
	}
}

// enrichTags converts a feature's tag keys to TagResponse using the provided map.
func enrichTags(keys []string, tagMap map[string]tag.Tag) []TagResponse {
	tags := make([]TagResponse, 0, len(keys))
	for _, k := range keys {
		if t, ok := tagMap[k]; ok {
			tags = append(tags, TagResponse{Key: t.Key, Name: t.Name, Color: t.Color})
		} else {
			tags = append(tags, TagResponse{Key: k, Name: k, Color: "#6B7280"})
		}
	}
	return tags
}

// ToFeatureResponse maps a domain feature to its list response DTO.
// tagMap enriches the tag keys with name and color; pass nil if not available.
// tiers provides tier refs resolved from packs; pass nil if not available.
// packs provides pack membership info; pass nil if not available.
func ToFeatureResponse(f *feature.Feature, tagMap map[string]tag.Tag, tiers []TierRef, packs ...[]PackRef) FeatureResponse {
	var packRefs []PackRef
	if len(packs) > 0 && packs[0] != nil {
		packRefs = packs[0]
	}
	if packRefs == nil {
		packRefs = []PackRef{}
	}
	tierRefs := tiers
	if tierRefs == nil {
		tierRefs = []TierRef{}
	}
	ruleCount := f.RuleCount
	if ruleCount == 0 {
		ruleCount = len(f.Rules)
	}
	return FeatureResponse{
		ID:             f.ID,
		Key:            f.Key,
		Name:           f.Name,
		Description:    f.Description,
		Enabled:        f.Enabled,
		ValueType:      string(f.ValueType),
		DefaultValue:   f.DefaultValue,
		ActiveFrom:     formatTimePtr(f.ActiveFrom),
		ActiveUntil:    formatTimePtr(f.ActiveUntil),
		Environments:   f.Environments,
		AccessPolicy:   string(f.AccessPolicy),
		AuthProfileKey: f.AuthProfileKey,
		InputContract:  toInputContractResponse(f.InputContract),
		Metadata:       f.Metadata,
		Tags:           enrichTags(f.Tags, tagMap),
		Packs:          packRefs,
		RolloutSalt:    f.RolloutSalt,
		RuleCount:      ruleCount,
		CreatedAt:      formatTime(f.CreatedAt),
		UpdatedAt:      formatTime(f.UpdatedAt),
		CreatedBy:      f.CreatedBy,
		UpdatedBy:      f.UpdatedBy,
		TrialUntil:     formatTimePtr(f.TrialUntil),
		TrialValue:     f.TrialValue,
		Tiers:          tierRefs,
	}
}

// ToFeatureSummaryResponse maps a domain feature to the lightweight list DTO.
func ToFeatureSummaryResponse(f *feature.Feature, tagMap map[string]tag.Tag, tiers []TierRef) FeatureSummaryResponse {
	tierRefs := tiers
	if tierRefs == nil {
		tierRefs = []TierRef{}
	}
	return FeatureSummaryResponse{
		ID:             f.ID,
		Key:            f.Key,
		Name:           f.Name,
		Description:    f.Description,
		Enabled:        f.Enabled,
		ValueType:      string(f.ValueType),
		Environments:   f.Environments,
		AccessPolicy:   string(f.AccessPolicy),
		AuthProfileKey: f.AuthProfileKey,
		Tags:           enrichTags(f.Tags, tagMap),
		PackCount:      f.PackCount,
		RuleCount:      f.RuleCount,
		CreatedAt:      formatTime(f.CreatedAt),
		UpdatedAt:      formatTime(f.UpdatedAt),
		CreatedBy:      f.CreatedBy,
		UpdatedBy:      f.UpdatedBy,
		TrialUntil:     formatTimePtr(f.TrialUntil),
		Tiers:          tierRefs,
	}
}

// ToFeatureDetailResponse maps a domain feature to its detail response DTO.
// tagMap enriches the tag keys with name and color; pass nil if not available.
// tiers provides tier refs resolved from packs; pass nil if not available.
// packs provides pack membership info; pass nil if not available.
func ToFeatureDetailResponse(f *feature.Feature, tagMap map[string]tag.Tag, tiers []TierRef, packs ...[]PackRef) FeatureDetailResponse {
	rules := make([]RuleResponse, 0, len(f.Rules))
	for i := range f.Rules {
		rules = append(rules, ToRuleResponse(&f.Rules[i]))
	}
	var packRefs []PackRef
	if len(packs) > 0 {
		packRefs = packs[0]
	}
	return FeatureDetailResponse{
		FeatureResponse: ToFeatureResponse(f, tagMap, tiers, packRefs),
		Rules:           rules,
	}
}

// ToRuleResponse maps a domain rule to its response DTO.
func ToRuleResponse(r *feature.Rule) RuleResponse {
	metadata := make(map[string]any, len(r.Metadata)+1)
	for k, v := range r.Metadata {
		metadata[k] = v
	}
	if metadata[engine.ConditionsBuilderMetadataKey] == nil && r.Expression != "" {
		if tree := engine.ParseToBuilderTree(r.Expression, r.SourceBindings); tree != nil {
			metadata[engine.ConditionsBuilderMetadataKey] = tree
		}
	}

	return RuleResponse{
		ID:                  r.ID,
		Name:                r.Name,
		Priority:            r.Priority,
		Enabled:             r.Enabled,
		Expression:          r.Expression,
		Value:               r.Value,
		RolloutPercentage:   r.RolloutPercentage,
		SourceBindings:      toSourceBindingsResponse(r.SourceBindings),
		ExternalAPIBindings: toExternalAPIBindingsResponse(r.ExternalAPIBindings),
		Metadata:            metadata,
		CreatedAt:           formatTime(r.CreatedAt),
		UpdatedAt:           formatTime(r.UpdatedAt),
	}
}

func toExternalAPIBindingsResponse(bindings []feature.ExternalAPIBinding) []ExternalAPIBindingResponse {
	result := make([]ExternalAPIBindingResponse, 0, len(bindings))
	for _, b := range bindings {
		mappings := make([]ParamMappingResponse, 0, len(b.ParamMappings))
		for _, m := range b.ParamMappings {
			mappings = append(mappings, ParamMappingResponse{
				ParamName:    m.ParamName,
				Mode:         m.Mode,
				InputPath:    m.InputPath,
				LiteralValue: m.LiteralValue,
			})
		}
		result = append(result, ExternalAPIBindingResponse{
			ExternalAPIKey: b.ExternalAPIKey,
			ParamMappings:  mappings,
			FailMode:       string(b.FailMode),
			CacheTTL:       b.CacheTTL,
		})
	}
	return result
}

func toInputContractResponse(contract feature.InputContract) InputContractResponse {
	headers := make([]InputHeaderResponse, 0, len(contract.Headers))
	for _, header := range contract.Headers {
		headers = append(headers, InputHeaderResponse{
			HeaderName:    header.HeaderName,
			ExpressionKey: header.ExpressionKey,
			Label:         header.Label,
			Type:          string(header.Type),
			Required:      header.Required,
			Description:   header.Description,
		})
	}

	return InputContractResponse{
		Headers:            headers,
		RequestBodyExample: contract.RequestBodyExample,
		RequestBodySchema:  contract.RequestBodySchema,
	}
}

func toSourceBindingsResponse(bindings feature.SourceBindings) SourceBindingsResponse {
	segments := make([]SegmentSourceBindingResponse, 0, len(bindings.Segments))
	for _, binding := range bindings.Segments {
		segments = append(segments, SegmentSourceBindingResponse{
			SegmentKey: binding.SegmentKey,
			LookupPath: binding.LookupPath,
		})
	}
	return SourceBindingsResponse{Segments: segments}
}

// ToSegmentResponse maps a domain segment to its response DTO.
func ToSegmentResponse(s *segment.Segment) SegmentResponse {
	return SegmentResponse{
		ID:            s.ID,
		Key:           s.Key,
		Name:          s.Name,
		Description:   s.Description,
		Metadata:      s.Metadata,
		RecordCount:   s.RecordCount,
		RecordKeyPath: s.RecordKeyPath,
		PreviewFields: s.PreviewFields,
		SourceType:    string(s.SourceType),
		LastImportAt:  formatTimePtr(s.LastImportAt),
		CreatedAt:     formatTime(s.CreatedAt),
		UpdatedAt:     formatTime(s.UpdatedAt),
		CreatedBy:     s.CreatedBy,
		UpdatedBy:     s.UpdatedBy,
	}
}

// ToSegmentSchemaResponse maps segment schema metadata to its response DTO.
func ToSegmentSchemaResponse(s *segment.Segment) SegmentSchemaResponse {
	return SegmentSchemaResponse{
		SegmentKey:           s.Key,
		Schema:               s.Schema,
		RecordKeyPath:        s.RecordKeyPath,
		ActiveDatasetVersion: s.ActiveDatasetVersion,
		PreviewFields:        s.PreviewFields,
		SourceType:           string(s.SourceType),
		LastImportAt:         formatTimePtr(s.LastImportAt),
		RecordCount:          s.RecordCount,
	}
}

// ToSegmentRecordResponse maps a domain segment record to its response DTO.
func ToSegmentRecordResponse(r *segment.Record) SegmentRecordResponse {
	return SegmentRecordResponse{
		ID:         r.ID,
		RecordKey:  r.RecordKey,
		Attributes: r.Attributes,
		CreatedAt:  formatTime(r.CreatedAt),
	}
}

// ToAuditErrorResponse maps a domain eval error to its response DTO.
func ToAuditErrorResponse(e *audit.EvalError) AuditErrorResponse {
	return AuditErrorResponse{
		ID:         e.ID,
		FeatureKey: e.FeatureKey,
		RuleID:     e.RuleID,
		ErrorType:  e.ErrorType,
		Message:    e.Message,
		TenantID:   e.TenantID,
		CampusID:   e.CampusID,
		ProgramID:  e.ProgramID,
		RequestID:  e.RequestID,
		CreatedAt:  formatTime(e.CreatedAt),
	}
}

// ToPackResponse maps a domain pack to its response DTO.
// tier provides the resolved tier ref; pass nil if the pack has no tier.
// resolvedFeatureCount is the total feature count including inherited packs.
func ToPackResponse(p *pack.Pack, tier *TierRef, resolvedFeatureCount int) PackResponse {
	featureKeys := p.FeatureKeys
	if featureKeys == nil {
		featureKeys = []string{}
	}
	inheritsFrom := p.InheritsFrom
	if inheritsFrom == nil {
		inheritsFrom = []string{}
	}
	return PackResponse{
		ID:                   p.ID,
		Key:                  p.Key,
		Name:                 p.Name,
		Description:          p.Description,
		FeatureKeys:          featureKeys,
		Enabled:              p.Enabled,
		Metadata:             p.Metadata,
		CreatedAt:            formatTime(p.CreatedAt),
		UpdatedAt:            formatTime(p.UpdatedAt),
		CreatedBy:            p.CreatedBy,
		UpdatedBy:            p.UpdatedBy,
		TierKey:              p.TierKey,
		Tier:                 tier,
		InheritsFrom:         inheritsFrom,
		TrialUntil:           formatTimePtr(p.TrialUntil),
		ResolvedFeatureCount: resolvedFeatureCount,
	}
}

// ToAuthProfileResponse maps an auth profile to its response DTO.
func ToAuthProfileResponse(profile *authprofile.Profile) AuthProfileResponse {
	return AuthProfileResponse{
		ID:              profile.ID,
		Key:             profile.Key,
		Name:            profile.Name,
		Active:          profile.Active,
		Type:            string(profile.Type),
		Config:          profile.Config,
		CacheTTLSeconds: profile.CacheTTLSeconds,
		Version:         profile.Version,
		HasSecret:       profile.HasSecret,
		CreatedAt:       formatTime(profile.CreatedAt),
		UpdatedAt:       formatTime(profile.UpdatedAt),
		CreatedBy:       profile.CreatedBy,
		UpdatedBy:       profile.UpdatedBy,
	}
}

// ToExternalAPIResponse maps a reusable external API to its response DTO.
func ToExternalAPIResponse(api *externalapi.ExternalAPI) ExternalAPIResponse {
	return ExternalAPIResponse{
		ID:                  api.ID,
		Key:                 api.Key,
		Name:                api.Name,
		Active:              api.Active,
		Request:             api.Request,
		Params:              api.Params,
		ExpressionVariables: api.ExpressionVariables,
		ResponseValidation:  api.ResponseValidation,
		HasSecrets:          api.HasSecrets,
		Version:             api.Version,
		CreatedAt:           formatTime(api.CreatedAt),
		UpdatedAt:           formatTime(api.UpdatedAt),
		CreatedBy:           api.CreatedBy,
		UpdatedBy:           api.UpdatedBy,
	}
}

// ToActivationResponse maps a domain pack activation to its response DTO.
func ToActivationResponse(a *pack.Activation) ActivationResponse {
	return ActivationResponse{
		ID:          a.ID,
		PackKey:     a.PackKey,
		TargetType:  string(a.TargetType),
		TargetID:    a.TargetID,
		ActivatedAt: formatTime(a.ActivatedAt),
		ActivatedBy: a.ActivatedBy,
		ExpiresAt:   formatTimePtr(a.ExpiresAt),
		Metadata:    a.Metadata,
	}
}

// PacksToRefs converts a slice of packs to a slice of PackRef.
func PacksToRefs(packs []pack.Pack) []PackRef {
	refs := make([]PackRef, 0, len(packs))
	for i := range packs {
		refs = append(refs, PackRef{Key: packs[i].Key, Name: packs[i].Name})
	}
	return refs
}

// ToTierRef maps a predefined tier.Def to a lightweight reference.
func ToTierRef(t *tier.Def) TierRef {
	return TierRef{
		Key:   t.Key,
		Name:  t.Name,
		Color: t.Color,
	}
}

// TiersToRefs maps a slice of predefined tier.Def to refs.
func TiersToRefs(tiers []tier.Def) []TierRef {
	refs := make([]TierRef, 0, len(tiers))
	for i := range tiers {
		refs = append(refs, ToTierRef(&tiers[i]))
	}
	return refs
}
