package middleware

import "testing"

func TestSanitizeLogField(t *testing.T) {
	in := "/api/v2/plugins/ok\nINJECTED\rline"
	got := sanitizeLogField(in)
	want := "/api/v2/plugins/ok?INJECTED?line"
	if got != want {
		t.Fatalf("sanitizeLogField(%q) = %q; want %q", in, got, want)
	}
	if sanitizeLogField("/plain/path") != "/plain/path" {
		t.Fatal("clean path should pass through unchanged")
	}
}
