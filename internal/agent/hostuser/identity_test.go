package hostuser

import (
	"strings"
	"testing"
)

func TestMarkerIsPasswdSafe(t *testing.T) {
	marker := Marker("0123456789abcdef0123456789abcdef")
	if marker != "WEBYCP-0123456789abcdef0123456789abcdef" {
		t.Fatalf("unexpected marker: %q", marker)
	}
	if strings.ContainsAny(marker, ":\n") {
		t.Fatalf("marker contains a passwd field delimiter: %q", marker)
	}
}
