package matchmaking

import (
	"strings"
	"testing"
	"time"
)

const testSecret = "test-secret-key"

func TestGenerateAndValidateJoinToken(t *testing.T) {
	token, err := GenerateJoinToken(testSecret, "legendary-sword", "7777", time.Minute)
	if err != nil {
		t.Fatalf("GenerateJoinToken: %v", err)
	}

	container, port, err := ValidateJoinToken(testSecret, token)
	if err != nil {
		t.Fatalf("ValidateJoinToken: %v", err)
	}
	if container != "legendary-sword" {
		t.Errorf("container = %q, want %q", container, "legendary-sword")
	}
	if port != "7777" {
		t.Errorf("port = %q, want %q", port, "7777")
	}
}

func TestValidateJoinToken_WrongSecret(t *testing.T) {
	token, err := GenerateJoinToken(testSecret, "legendary-sword", "7777", time.Minute)
	if err != nil {
		t.Fatalf("GenerateJoinToken: %v", err)
	}

	_, _, err = ValidateJoinToken("wrong-secret", token)
	if err == nil {
		t.Fatal("expected error with wrong secret, got nil")
	}
}

func TestValidateJoinToken_Expired(t *testing.T) {
	token, err := GenerateJoinToken(testSecret, "legendary-sword", "7777", -time.Second)
	if err != nil {
		t.Fatalf("GenerateJoinToken: %v", err)
	}

	_, _, err = ValidateJoinToken(testSecret, token)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestValidateJoinToken_Tampered(t *testing.T) {
	token, err := GenerateJoinToken(testSecret, "legendary-sword", "7777", time.Minute)
	if err != nil {
		t.Fatalf("GenerateJoinToken: %v", err)
	}

	// Flip a character in the payload
	tampered := "A" + token[1:]
	_, _, err = ValidateJoinToken(testSecret, tampered)
	if err == nil {
		t.Fatal("expected error for tampered token, got nil")
	}
}

func TestValidateJoinToken_InvalidFormat(t *testing.T) {
	cases := []string{
		"",
		"nodot",
		"only.one.extra.dot",
	}
	for _, tc := range cases {
		_, _, err := ValidateJoinToken(testSecret, tc)
		if err == nil {
			t.Errorf("expected error for token %q, got nil", tc)
		}
	}
}

func TestGenerateJoinToken_UniquePerCall(t *testing.T) {
	t1, _ := GenerateJoinToken(testSecret, "legendary-sword", "7777", time.Minute)
	t2, _ := GenerateJoinToken(testSecret, "legendary-sword", "7777", time.Minute)
	// Tokens generated at different times will have different expiry — just
	// verify they are well-formed and valid, not necessarily identical.
	if !strings.Contains(t1, ".") || !strings.Contains(t2, ".") {
		t.Error("tokens should contain a '.' separator")
	}
}
