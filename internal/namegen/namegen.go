package namegen

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

var (
	// Adjectives - video game themed descriptors
	adjectives = []string{
		"legendary", "ancient", "mystic", "brave", "shadow", "fierce",
		"crimson", "silver", "iron", "golden", "dark", "holy",
		"cursed", "blessed", "frozen", "burning", "ethereal", "phantom",
		"storm", "dragon", "thunder", "void", "celestial", "arcane",
		"corrupted", "pure", "savage", "noble", "rogue", "eternal",
		"forgotten", "enchanted", "demon", "angel", "blood", "crystal",
		"chaos", "order", "primal", "astral", "wild", "divine",
	}

	// Nouns - video game items, characters, and creatures
	nouns = []string{
		"sword", "shield", "tome", "paladin", "rogue", "barbarian",
		"dragon", "phoenix", "golem", "wizard", "archer", "knight",
		"blade", "axe", "staff", "crown", "helm", "gauntlet",
		"wyrm", "griffin", "hydra", "titan", "specter", "wraith",
		"sentinel", "champion", "warden", "guardian", "slayer", "hunter",
		"reaper", "oracle", "sage", "monk", "warlock", "crusader",
		"berserker", "assassin", "druid", "necromancer", "sorcerer", "ranger",
		"templar", "valkyrie", "samurai", "ninja", "ronin", "shogun",
	}
)

// Generate creates a random video game themed server name
// Format: adjective-noun (e.g., "legendary-sword", "shadow-dragon")
func Generate() (string, error) {
	adjIndex, err := secureRandomInt(len(adjectives))
	if err != nil {
		return "", fmt.Errorf("failed to generate random adjective: %w", err)
	}

	nounIndex, err := secureRandomInt(len(nouns))
	if err != nil {
		return "", fmt.Errorf("failed to generate random noun: %w", err)
	}

	return fmt.Sprintf("%s-%s", adjectives[adjIndex], nouns[nounIndex]), nil
}

// secureRandomInt generates a cryptographically secure random integer in the range [0, max)
func secureRandomInt(max int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}
