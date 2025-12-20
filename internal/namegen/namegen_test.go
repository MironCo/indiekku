package namegen

import (
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	name, err := Generate()
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	// Check format: should be "word-word"
	parts := strings.Split(name, "-")
	if len(parts) != 2 {
		t.Errorf("Expected name format 'adjective-noun', got: %s", name)
	}

	// Check that both parts are non-empty
	if parts[0] == "" || parts[1] == "" {
		t.Errorf("Name parts should not be empty: %s", name)
	}

	// Verify the adjective is in the list
	found := false
	for _, adj := range adjectives {
		if parts[0] == adj {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Adjective '%s' not found in adjectives list", parts[0])
	}

	// Verify the noun is in the list
	found = false
	for _, noun := range nouns {
		if parts[1] == noun {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Noun '%s' not found in nouns list", parts[1])
	}
}

func TestGenerateUniqueness(t *testing.T) {
	// Generate multiple names and check that we get some variety
	names := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		name, err := Generate()
		if err != nil {
			t.Fatalf("Generate() returned error on iteration %d: %v", i, err)
		}
		names[name] = true
	}

	// With our word lists, we should see at least some unique names
	// (We have 43 adjectives * 48 nouns = 2,064 possible combinations)
	if len(names) < 50 {
		t.Errorf("Expected at least 50 unique names out of %d generations, got %d", iterations, len(names))
	}
}

func TestSecureRandomInt(t *testing.T) {
	max := 10
	for i := 0; i < 100; i++ {
		n, err := secureRandomInt(max)
		if err != nil {
			t.Fatalf("secureRandomInt() returned error: %v", err)
		}
		if n < 0 || n >= max {
			t.Errorf("secureRandomInt(%d) returned %d, expected value in range [0, %d)", max, n, max)
		}
	}
}
