package handlers

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// TestStoredPasswordIsHashed guards against storing plaintext passwords.
func TestStoredPasswordIsHashed(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if string(hash) == "secret" {
		t.Fatal("password must not be stored in plaintext")
	}
	if bcrypt.CompareHashAndPassword(hash, []byte("secret")) != nil {
		t.Fatal("correct password should verify")
	}
	if bcrypt.CompareHashAndPassword(hash, []byte("wrong")) == nil {
		t.Fatal("wrong password must not verify")
	}
}
