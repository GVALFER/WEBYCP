package envx

import "testing"

func TestString(t *testing.T) {
	t.Setenv("WEBYCP_TEST_VALUE", " configured ")

	if value := String("WEBYCP_TEST_VALUE", "fallback"); value != "configured" {
		t.Fatalf("String() = %q, want configured", value)
	}
}

func TestBool(t *testing.T) {
	t.Setenv("WEBYCP_TEST_BOOL", "true")
	if !Bool("WEBYCP_TEST_BOOL", false) {
		t.Fatal("expected true")
	}

	t.Setenv("WEBYCP_TEST_BOOL", "invalid")
	if !Bool("WEBYCP_TEST_BOOL", true) {
		t.Fatal("expected fallback")
	}
}

func TestStringFallback(t *testing.T) {
	t.Setenv("WEBYCP_TEST_VALUE", " ")

	if value := String("WEBYCP_TEST_VALUE", "fallback"); value != "fallback" {
		t.Fatalf("String() = %q, want fallback", value)
	}
}
