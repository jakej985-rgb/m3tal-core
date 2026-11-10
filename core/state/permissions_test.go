package state

import "testing"

func TestNormalizePermissionsValid(t *testing.T) {
	perms, err := NormalizePermissions([]string{ScopeSystemRead, ScopePluginsWrite})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(perms) != 2 || perms[0] != ScopeSystemRead || perms[1] != ScopePluginsWrite {
		t.Fatalf("unexpected perms: %v", perms)
	}
}

func TestNormalizePermissionsRejectsInvalid(t *testing.T) {
	// A typo'd or attacker-supplied scope must never silently become "*".
	for _, input := range [][]string{
		{},
		{"system;read"},
		{"bogus:scope"},
		{""},
		{"  "},
		{ScopeSystemRead, "not-a-scope"},
	} {
		perms, err := NormalizePermissions(input)
		if err == nil {
			t.Errorf("NormalizePermissions(%v) = %v, expected error", input, perms)
		}
		for _, p := range perms {
			if p == ScopeFull {
				t.Errorf("NormalizePermissions(%v) returned full access", input)
			}
		}
	}
}

func TestNormalizePermissionsDedupes(t *testing.T) {
	perms, err := NormalizePermissions([]string{ScopeFull, ScopeFull, " * "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(perms) != 1 || perms[0] != ScopeFull {
		t.Fatalf("unexpected perms: %v", perms)
	}
}
