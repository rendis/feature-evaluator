package tier

import "time"

// MaxIconSize is the maximum allowed icon file size (32KB).
const MaxIconSize = 32 * 1024

// BuiltinIcons lists the available built-in icon keys.
var BuiltinIcons = []string{
	"crown", "star", "diamond", "shield", "rocket", "lightning",
	"gem", "fire", "lock", "check-circle", "zap", "trophy",
}

// Tier represents a subscription plan level.
type Tier struct {
	ID           string    `json:"id"`
	WorkspaceKey string    `json:"workspaceKey"`
	Key          string    `json:"key"`
	Name         string    `json:"name"`
	Level        int       `json:"level"`
	Color        string    `json:"color"`
	Icon         string    `json:"icon"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	CreatedBy    string    `json:"createdBy"`
}

// TierIcon represents a custom uploaded icon.
type TierIcon struct {
	ID           string    `json:"id"`
	WorkspaceKey string    `json:"workspaceKey"`
	Name         string    `json:"name"`
	ContentType  string    `json:"contentType"`
	Data         []byte    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
	CreatedBy    string    `json:"createdBy"`
}
