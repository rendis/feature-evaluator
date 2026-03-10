package member

// Permission represents a granular permission.
type Permission string

// All supported permissions.
const (
	PermFeaturesRead      Permission = "features.read"
	PermFeaturesWrite     Permission = "features.write"
	PermSegmentsRead      Permission = "segments.read"
	PermSegmentsWrite     Permission = "segments.write"
	PermMembersRead       Permission = "members.read"
	PermMembersManage     Permission = "members.manage"
	PermSettingsManage    Permission = "settings.manage"
	PermPacksRead         Permission = "packs.read"
	PermPacksWrite        Permission = "packs.write"
	PermExperimentsRead   Permission = "experiments.read"
	PermExperimentsWrite  Permission = "experiments.write"
	PermAuditRead         Permission = "audit.read"
	PermWorkspaceDelete   Permission = "workspace.delete"
	PermOwnershipTransfer Permission = "ownership.transfer"
)

// rolePermissions maps each role to its set of permissions.
var rolePermissions = map[Role][]Permission{
	RoleOwner: {
		PermFeaturesRead, PermFeaturesWrite,
		PermSegmentsRead, PermSegmentsWrite,
		PermPacksRead, PermPacksWrite,
		PermExperimentsRead, PermExperimentsWrite,
		PermMembersRead, PermMembersManage,
		PermSettingsManage, PermAuditRead,
		PermWorkspaceDelete, PermOwnershipTransfer,
	},
	RoleAdmin: {
		PermFeaturesRead, PermFeaturesWrite,
		PermSegmentsRead, PermSegmentsWrite,
		PermPacksRead, PermPacksWrite,
		PermExperimentsRead, PermExperimentsWrite,
		PermMembersRead, PermMembersManage,
		PermSettingsManage, PermAuditRead,
	},
	RoleEditor: {
		PermFeaturesRead, PermFeaturesWrite,
		PermSegmentsRead, PermSegmentsWrite,
		PermPacksRead, PermPacksWrite,
		PermExperimentsRead, PermExperimentsWrite,
		PermMembersRead, PermAuditRead,
	},
	RoleViewer: {
		PermFeaturesRead, PermSegmentsRead,
		PermPacksRead, PermExperimentsRead,
		PermMembersRead, PermAuditRead,
	},
}

// HasPermission checks whether a role has a specific permission.
func HasPermission(role Role, perm Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

// GetPermissions returns all permissions for a given role.
func GetPermissions(role Role) []Permission {
	return rolePermissions[role]
}

// AllowedAPIKeyPermissions is the set of permissions that can be assigned to admin API keys.
var AllowedAPIKeyPermissions = map[Permission]bool{
	PermFeaturesRead:     true,
	PermFeaturesWrite:    true,
	PermSegmentsRead:     true,
	PermSegmentsWrite:    true,
	PermPacksRead:        true,
	PermPacksWrite:       true,
	PermExperimentsRead:  true,
	PermExperimentsWrite: true,
	PermAuditRead:        true,
}

// ForbiddenAPIKeyPrefixes lists permission prefixes that are never allowed on API keys.
var ForbiddenAPIKeyPrefixes = []string{
	"members.",
	"settings.",
	"workspace.",
	"ownership.",
}

// IsAllowedAPIKeyPermission checks whether a permission string is allowed for admin API keys.
func IsAllowedAPIKeyPermission(perm string) bool {
	return AllowedAPIKeyPermissions[Permission(perm)]
}
