package crypto

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("hello world — sensitive credentials")

	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	decrypted, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("got %q, want %q", decrypted, plaintext)
	}
}

func TestDifferentCiphertexts(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("same input")

	ct1, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	ct2, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(ct1, ct2) {
		t.Fatal("expected different ciphertexts for same plaintext (random nonce)")
	}
}

func TestWrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	if _, err := rand.Read(key1); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(key2); err != nil {
		t.Fatal(err)
	}

	ciphertext, err := Encrypt(key1, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = Decrypt(key2, ciphertext)
	if err == nil {
		t.Fatal("expected error decrypting with wrong key")
	}
}

func TestInvalidKeyLength(t *testing.T) {
	shortKey := make([]byte, 16)

	_, err := Encrypt(shortKey, []byte("data"))
	if err != ErrInvalidKeyLength {
		t.Fatalf("Encrypt: got %v, want ErrInvalidKeyLength", err)
	}

	_, err = Decrypt(shortKey, []byte("data"))
	if err != ErrInvalidKeyLength {
		t.Fatalf("Decrypt: got %v, want ErrInvalidKeyLength", err)
	}
}

func TestCiphertextTooShort(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	_, err := Decrypt(key, []byte("short"))
	if err != ErrCiphertextTooShort {
		t.Fatalf("got %v, want ErrCiphertextTooShort", err)
	}
}
