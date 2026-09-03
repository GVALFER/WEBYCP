package validate

import (
	"strings"
	"testing"
)

func TestDomain(t *testing.T) {
	tests := map[string]string{
		"Example.COM.":    "example.com",
		"münich.example":  "xn--mnich-kva.example",
		"sub.example.com": "sub.example.com",
	}
	for input, expected := range tests {
		actual, err := Domain(input)
		if err != nil {
			t.Fatalf("Domain(%q): %v", input, err)
		}
		if actual != expected {
			t.Fatalf("Domain(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestDomainRejectsInvalidNames(t *testing.T) {
	for _, value := range []string{
		"localhost", "127.0.0.1", "-example.com", "example-.com",
		"example..com", "*.example.com", "example.com/path", "under_score.example",
	} {
		if _, err := Domain(value); err == nil {
			t.Fatalf("Domain(%q) should fail", value)
		}
	}
}

func TestDomainAliasesValidatesAndSorts(t *testing.T) {
	aliases, err := DomainAliases("example.com", []string{"www.example.net", "cdn.example.net"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(aliases, ",") != "cdn.example.net,www.example.net" {
		t.Fatalf("aliases = %#v", aliases)
	}
	for _, values := range [][]string{
		{"example.com"}, {"www.example.net", "www.example.net"}, {"Example.net"},
	} {
		if _, err := DomainAliases("example.com", values); err == nil {
			t.Fatalf("expected aliases %#v to be rejected", values)
		}
	}
}
