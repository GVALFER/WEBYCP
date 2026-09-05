package backupfmt

import "errors"

var (
	ErrVersion = errors.New("This backup format is not supported. Create a new backup with the current version of WEBYCP")
	ErrInvalid = errors.New("The backup archive is invalid or damaged. Use another verified backup")
)
