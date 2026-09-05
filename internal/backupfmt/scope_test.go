package backupfmt

import "testing"

func TestValidateScope(t *testing.T) {
	for _, test := range []struct {
		name                              string
		manifest                          Manifest
		files, databases, metadata, valid bool
	}{
		{name: "empty", manifest: Manifest{Files: true}},
		{name: "missing files", files: true},
		{name: "missing databases", databases: true},
		{name: "missing metadata", metadata: true},
		{name: "partial", manifest: Manifest{Files: true, Metadata: true}, files: true, valid: true},
		{name: "full", manifest: Manifest{Files: true, Databases: true, Metadata: true}, files: true, databases: true, metadata: true, valid: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.manifest.ValidateScope(test.files, test.databases, test.metadata)
			if (err == nil) != test.valid {
				t.Fatalf("ValidateScope() = %v, want valid %v", err, test.valid)
			}
		})
	}
}
