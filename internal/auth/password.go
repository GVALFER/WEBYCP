package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	passwordMemory  = 64 * 1024
	passwordTime    = 3
	passwordThreads = 2
	passwordSaltLen = 16
	passwordKeyLen  = 32
)

func HashPassword(password string) (string, error) {
	salt := make([]byte, passwordSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, passwordTime, passwordMemory, passwordThreads, passwordKeyLen)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		passwordMemory,
		passwordTime,
		passwordThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func VerifyPassword(password, encoded string) bool {
	value, ok := parsePasswordHash(encoded)
	if !ok {
		return false
	}
	actual := argon2.IDKey([]byte(password), value.salt, value.iterations, value.memory, value.threads, uint32(len(value.key)))
	return subtle.ConstantTimeCompare(actual, value.key) == 1
}

// ValidPasswordHash checks the encoding and cost bounds without running Argon2.
func ValidPasswordHash(encoded string) bool {
	_, ok := parsePasswordHash(encoded)
	return ok
}

type passwordHash struct {
	memory, iterations uint32
	threads            uint8
	salt, key          []byte
}

func parsePasswordHash(encoded string) (passwordHash, bool) {
	var value passwordHash
	if len(encoded) > 128 {
		return value, false
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return value, false
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return value, false
	}

	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &value.memory, &value.iterations, &value.threads); err != nil {
		return value, false
	}
	if value.memory == 0 || value.memory > passwordMemory || value.iterations == 0 || value.iterations > passwordTime || value.threads == 0 || value.threads > passwordThreads {
		return value, false
	}
	if parts[2] != fmt.Sprintf("v=%d", argon2.Version) || parts[3] != fmt.Sprintf("m=%d,t=%d,p=%d", value.memory, value.iterations, value.threads) {
		return value, false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) != passwordSaltLen || base64.RawStdEncoding.EncodeToString(salt) != parts[4] {
		return value, false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expected) != passwordKeyLen || base64.RawStdEncoding.EncodeToString(expected) != parts[5] {
		return value, false
	}
	value.salt, value.key = salt, expected
	return value, true
}

func validatePasswordHash(encoded string) error {
	if !strings.HasPrefix(encoded, "$argon2id$") {
		return errors.New("unsupported password hash")
	}

	return nil
}
