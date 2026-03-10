package tier

// TierDef represents a predefined tier.
type TierDef struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Color    string `json:"color"`
	Icon     string `json:"icon"`
	Category string `json:"category"`
}

// AllTiers is the hardcoded catalog of 24 predefined tiers.
var AllTiers = []TierDef{
	// Entry/Access
	{Key: "free", Name: "Free", Color: "#6B7280", Icon: "tier-free", Category: "entry"},
	{Key: "starter", Name: "Starter", Color: "#64748B", Icon: "tier-starter", Category: "entry"},
	{Key: "hobby", Name: "Hobby", Color: "#78716C", Icon: "tier-hobby", Category: "entry"},
	{Key: "community", Name: "Community", Color: "#71717A", Icon: "tier-community", Category: "entry"},
	{Key: "lite", Name: "Lite", Color: "#9CA3AF", Icon: "tier-lite", Category: "entry"},

	// Growth
	{Key: "basic", Name: "Basic", Color: "#3B82F6", Icon: "tier-basic", Category: "growth"},
	{Key: "standard", Name: "Standard", Color: "#2563EB", Icon: "tier-standard", Category: "growth"},
	{Key: "plus", Name: "Plus", Color: "#0EA5E9", Icon: "tier-plus", Category: "growth"},
	{Key: "growth", Name: "Growth", Color: "#06B6D4", Icon: "tier-growth", Category: "growth"},

	// Advanced
	{Key: "pro", Name: "Pro", Color: "#8B5CF6", Icon: "tier-pro", Category: "advanced"},
	{Key: "professional", Name: "Professional", Color: "#7C3AED", Icon: "tier-professional", Category: "advanced"},
	{Key: "business", Name: "Business", Color: "#6366F1", Icon: "tier-business", Category: "advanced"},
	{Key: "premium", Name: "Premium", Color: "#A855F7", Icon: "tier-premium", Category: "advanced"},

	// Top Level
	{Key: "enterprise", Name: "Enterprise", Color: "#F59E0B", Icon: "tier-enterprise", Category: "top"},
	{Key: "ultimate", Name: "Ultimate", Color: "#EAB308", Icon: "tier-ultimate", Category: "top"},
	{Key: "elite", Name: "Elite", Color: "#D97706", Icon: "tier-elite", Category: "top"},
	{Key: "corporate", Name: "Corporate", Color: "#CA8A04", Icon: "tier-corporate", Category: "top"},

	// Specials
	{Key: "founder", Name: "Founder", Color: "#10B981", Icon: "tier-founder", Category: "special"},
	{Key: "early-adopter", Name: "Early Adopter", Color: "#14B8A6", Icon: "tier-early-adopter", Category: "special"},
	{Key: "legacy", Name: "Legacy", Color: "#059669", Icon: "tier-legacy", Category: "special"},
	{Key: "non-profit", Name: "Non-Profit", Color: "#F472B6", Icon: "tier-non-profit", Category: "special"},

	// Technical/Staff
	{Key: "internal", Name: "Internal", Color: "#EF4444", Icon: "tier-internal", Category: "technical"},
	{Key: "test", Name: "Test", Color: "#F97316", Icon: "tier-test", Category: "technical"},
	{Key: "beta", Name: "Beta", Color: "#FB923C", Icon: "tier-beta", Category: "technical"},
	{Key: "alpha", Name: "Alpha", Color: "#FBBF24", Icon: "tier-alpha", Category: "technical"},
	{Key: "staging", Name: "Staging", Color: "#A3E635", Icon: "tier-staging", Category: "technical"},
	{Key: "admin", Name: "Admin", Color: "#DC2626", Icon: "tier-admin", Category: "technical"},
}

// tierByKey provides O(1) lookup by key.
var tierByKey map[string]*TierDef

func init() {
	tierByKey = make(map[string]*TierDef, len(AllTiers))
	for i := range AllTiers {
		tierByKey[AllTiers[i].Key] = &AllTiers[i]
	}
}

// FindByKey returns the predefined tier with the given key, or nil if not found.
func FindByKey(key string) *TierDef {
	return tierByKey[key]
}

// FindByKeys returns the predefined tiers matching the given keys.
func FindByKeys(keys []string) []TierDef {
	result := make([]TierDef, 0, len(keys))
	for _, k := range keys {
		if td, ok := tierByKey[k]; ok {
			result = append(result, *td)
		}
	}
	return result
}

// ValidKey returns true if the key matches a predefined tier.
func ValidKey(key string) bool {
	_, ok := tierByKey[key]
	return ok
}
