package crypto

import (
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}

	tests := []string{
		"hello world",
		"",
		"a very long password with special chars: !@#$%^&*()",
		"unicode: 日本語テスト",
	}

	for _, plaintext := range tests {
		ciphertext, err := enc.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encrypt(%q): %v", plaintext, err)
		}

		if ciphertext == plaintext && plaintext != "" {
			t.Errorf("ciphertext should differ from plaintext")
		}

		decrypted, err := enc.Decrypt(ciphertext)
		if err != nil {
			t.Fatalf("Decrypt: %v", err)
		}

		if decrypted != plaintext {
			t.Errorf("got %q, want %q", decrypted, plaintext)
		}
	}
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	key := make([]byte, 32)
	enc, _ := NewEncryptor(key)

	c1, _ := enc.Encrypt("same")
	c2, _ := enc.Encrypt("same")

	if c1 == c2 {
		t.Error("two encryptions of the same plaintext should produce different ciphertexts (random nonce)")
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	key := make([]byte, 32)
	enc, _ := NewEncryptor(key)

	ciphertext, _ := enc.Encrypt("secret")

	// Tamper with the ciphertext
	tampered := []byte(ciphertext)
	tampered[len(tampered)-2] ^= 0xff

	_, err := enc.Decrypt(string(tampered))
	if err == nil {
		t.Error("expected error when decrypting tampered ciphertext")
	}
}

func TestNewEncryptorInvalidKeySize(t *testing.T) {
	_, err := NewEncryptor([]byte("too short"))
	if err == nil {
		t.Error("expected error for invalid key size")
	}
}

func TestDeriveKey(t *testing.T) {
	key1, err := DeriveKey("my-jwt-secret-that-is-long-enough")
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}

	if len(key1) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(key1))
	}

	// Same input produces same key
	key2, _ := DeriveKey("my-jwt-secret-that-is-long-enough")
	if string(key1) != string(key2) {
		t.Error("same input should produce same derived key")
	}

	// Different input produces different key
	key3, _ := DeriveKey("different-secret-value-here-long")
	if string(key1) == string(key3) {
		t.Error("different inputs should produce different keys")
	}
}
