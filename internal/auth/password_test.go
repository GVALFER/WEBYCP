package auth

import (
	"strings"
	"testing"
)

func TestPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}

	if !VerifyPassword("correct horse battery staple", hash) {
		t.Fatal("VerifyPassword() = false, want true")
	}
	if VerifyPassword("wrong password", hash) {
		t.Fatal("VerifyPassword() = true for wrong password")
	}
}

func TestPasswordHashValidation(t *testing.T) {
	hash, err := HashPassword("hash validation fixture")
	if err != nil {
		t.Fatal(err)
	}
	if !ValidPasswordHash(hash) {
		t.Fatal("generated password hash rejected")
	}
	for _, value := range []string{
		"plaintext", hash + "\n", "prefix" + hash,
		strings.Replace(hash, "p=2$", "p=2:injected$", 1),
		strings.Replace(hash, "m=65536", "m=99999999", 1),
		strings.Replace(hash, "v=19", "v=19junk", 1),
	} {
		if ValidPasswordHash(value) || VerifyPassword("hash validation fixture", value) {
			t.Fatal("invalid password hash accepted")
		}
	}
}
