package backupfmt

import "github.com/GVALFER/WEBYCP/internal/validate"

// ValidateScope rejects empty selections and content absent from the archive.
func (m Manifest) ValidateScope(files, databases, metadata bool) error {
	if !files && !databases && !metadata {
		return &validate.Error{Field: "scope", Message: "Select at least one restore scope"}
	}
	if files && !m.Files || databases && !m.Databases || metadata && !m.Metadata {
		return &validate.Error{Field: "scope", Message: "The selected content is not present in this backup"}
	}
	return nil
}
