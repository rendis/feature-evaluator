package member

import "testing"

func TestSecurityManagePermissionIsOwnerOnly(t *testing.T) {
	t.Parallel()

	if !HasPermission(RoleOwner, PermSecurityManage) {
		t.Fatal("RoleOwner missing PermSecurityManage")
	}
	if HasPermission(RoleAdmin, PermSecurityManage) {
		t.Fatal("RoleAdmin has PermSecurityManage, want denied")
	}
	if HasPermission(RoleEditor, PermSecurityManage) {
		t.Fatal("RoleEditor has PermSecurityManage, want denied")
	}
	if HasPermission(RoleViewer, PermSecurityManage) {
		t.Fatal("RoleViewer has PermSecurityManage, want denied")
	}
}

func TestSecurityManagePermissionIsForbiddenForAPIKeys(t *testing.T) {
	t.Parallel()

	if IsAllowedAPIKeyPermission(string(PermSecurityManage)) {
		t.Fatal("IsAllowedAPIKeyPermission(security.manage) = true, want false")
	}
}
