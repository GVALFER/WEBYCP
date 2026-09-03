package auth

import "testing"

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
