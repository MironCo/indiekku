package matchmaking

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// joinTokenPayload is the data embedded in a join token
type joinTokenPayload struct {
	ContainerName string `json:"c"`
	Port          string `json:"p"`
	ExpiresAt     int64  `json:"exp"`
}

// GenerateJoinToken creates a short-lived HMAC-SHA256 signed token scoped to
// one player connecting to one server. The game server should validate this
// on connect by calling ValidateJoinToken.
func GenerateJoinToken(secret, containerName, port string, ttl time.Duration) (string, error) {
	payload := joinTokenPayload{
		ContainerName: containerName,
		Port:          port,
		ExpiresAt:     time.Now().Add(ttl).Unix(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	sig := sign(secret, encodedPayload)

	return encodedPayload + "." + sig, nil
}

// ValidateJoinToken verifies the token signature and expiry.
// Returns the container name and port if valid.
func ValidateJoinToken(secret, token string) (containerName, port string, err error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid token format")
	}

	encodedPayload, providedSig := parts[0], parts[1]

	// Verify signature
	expectedSig := sign(secret, encodedPayload)
	if !hmac.Equal([]byte(expectedSig), []byte(providedSig)) {
		return "", "", fmt.Errorf("invalid token signature")
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return "", "", fmt.Errorf("invalid token encoding")
	}

	var payload joinTokenPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return "", "", fmt.Errorf("invalid token payload")
	}

	// Check expiry
	if time.Now().Unix() > payload.ExpiresAt {
		return "", "", fmt.Errorf("token expired")
	}

	return payload.ContainerName, payload.Port, nil
}

func sign(secret, data string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
