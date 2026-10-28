package security

import "testing"

func TestEncryptAndDecryptData(t *testing.T) {
	secret := "super-secret-db-password"
	original := "sensitive-database-payload-123"

	encrypted, err := EncryptData(original, secret)
	if err != nil {
		t.Fatalf("EncryptData error: %v", err)
	}
	if encrypted == original {
		t.Fatal("ciphertext must differ from plaintext")
	}

	decrypted, err := DecryptData(encrypted, secret)
	if err != nil {
		t.Fatalf("DecryptData error: %v", err)
	}
	if decrypted != original {
		t.Fatalf("decrypted = %q, want %q", decrypted, original)
	}

	if _, err := DecryptData(encrypted, "wrong-password"); err == nil {
		t.Fatal("expected error when decrypting with wrong password")
	}
}
