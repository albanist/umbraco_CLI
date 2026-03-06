package validate

import "testing"

func TestPathRejectsTraversal(t *testing.T) {
	if err := Path("images/../secret.png"); err == nil {
		t.Fatalf("expected traversal path to fail")
	}
}

func TestPathRejectsAbsolute(t *testing.T) {
	if err := Path("/etc/passwd"); err == nil {
		t.Fatalf("expected absolute path to fail")
	}
}

func TestStringRejectsControlChars(t *testing.T) {
	if err := String("hello\x07world"); err == nil {
		t.Fatalf("expected control chars to fail")
	}
}

func TestResourceIDRejectsQueryChars(t *testing.T) {
	if err := ResourceID("abc?fields=name"); err == nil {
		t.Fatalf("expected invalid id to fail")
	}
}

func TestNoPreEncodingRejectsEncodedValue(t *testing.T) {
	if err := NoPreEncoding("hello%20world"); err == nil {
		t.Fatalf("expected encoded input to fail")
	}
}

func TestInputValidatesRecursively(t *testing.T) {
	input := map[string]any{
		"parent": map[string]any{
			"id": "abc#123",
		},
	}
	if err := Input(input); err == nil {
		t.Fatalf("expected nested invalid id to fail")
	}
}

func TestInputAllowsSafeNestedValues(t *testing.T) {
	input := map[string]any{
		"parent": map[string]any{"id": "abc-123"},
		"values": []any{
			map[string]any{"alias": "title", "value": "Hello world"},
			map[string]any{"alias": "slug", "value": "hello-world"},
		},
	}
	if err := Input(input); err != nil {
		t.Fatalf("expected safe input to pass: %v", err)
	}
}
