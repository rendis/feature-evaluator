package tag

import "time"

// Tag represents a feature tag with a display color.
type Tag struct {
	ID           string    `json:"id"`
	WorkspaceKey string    `json:"workspaceKey"`
	Key          string    `json:"key"`
	Name         string    `json:"name"`
	Color        string    `json:"color"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	CreatedBy    string    `json:"createdBy"`
}

// Palette defines the 12 default tag colors assigned round-robin to migrated tags.
var Palette = []string{
	"#EF4444", // red
	"#F97316", // orange
	"#F59E0B", // amber
	"#84CC16", // lime
	"#22C55E", // green
	"#14B8A6", // teal
	"#3B82F6", // blue
	"#6366F1", // indigo
	"#A855F7", // purple
	"#EC4899", // pink
	"#F43F5E", // rose
	"#6B7280", // gray
}
