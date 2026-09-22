package password

import (
	"testing"
)

func TestPasswordHashing(t *testing.T) {
	plain := "password123"
	hash, err := Hash(plain)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if !Check(plain, hash) {
		t.Fatalf("expected password check to succeed")
	}

	if Check("wrongpassword", hash) {
		t.Fatalf("expected wrong password to fail check")
	}
}
