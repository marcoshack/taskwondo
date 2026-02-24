package model

import "testing"

func TestHasPermission_EmptyPermissions(t *testing.T) {
	info := &AuthInfo{Permissions: nil}
	for _, method := range []string{"GET", "HEAD", "OPTIONS", "POST", "PUT", "PATCH", "DELETE"} {
		if !info.HasPermission(method) {
			t.Errorf("empty permissions should allow %s", method)
		}
	}

	info2 := &AuthInfo{Permissions: []string{}}
	for _, method := range []string{"GET", "POST", "DELETE"} {
		if !info2.HasPermission(method) {
			t.Errorf("empty slice permissions should allow %s", method)
		}
	}
}

func TestHasPermission_ReadOnly(t *testing.T) {
	info := &AuthInfo{Permissions: []string{PermissionRead}}

	allowed := []string{"GET", "HEAD", "OPTIONS"}
	for _, method := range allowed {
		if !info.HasPermission(method) {
			t.Errorf("read permission should allow %s", method)
		}
	}

	denied := []string{"POST", "PUT", "PATCH", "DELETE"}
	for _, method := range denied {
		if info.HasPermission(method) {
			t.Errorf("read permission should deny %s", method)
		}
	}
}

func TestHasPermission_WriteAllowsAll(t *testing.T) {
	info := &AuthInfo{Permissions: []string{PermissionWrite}}
	for _, method := range []string{"GET", "HEAD", "OPTIONS", "POST", "PUT", "PATCH", "DELETE"} {
		if !info.HasPermission(method) {
			t.Errorf("write permission should allow %s", method)
		}
	}
}

func TestHasPermission_ReadAndWrite(t *testing.T) {
	info := &AuthInfo{Permissions: []string{PermissionRead, PermissionWrite}}
	for _, method := range []string{"GET", "POST", "DELETE"} {
		if !info.HasPermission(method) {
			t.Errorf("read+write permissions should allow %s", method)
		}
	}
}
