package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"github.com/MironCo/indiekku/internal/client"
	"github.com/MironCo/indiekku/internal/matchmaking"
	"github.com/MironCo/indiekku/internal/security"

	"github.com/gin-gonic/gin"
)

func main() {
	port := flag.String("port", "7070", "Port to listen on (0.0.0.0)")
	publicIP := flag.String("public-ip", "", "Public IP returned to clients (required)")
	maxPlayers := flag.Int("max-players", 8, "Max players per server before a new one is started")
	tokenSecret := flag.String("token-secret", "", "Secret for signing join tokens (auto-generated if empty)")
	indiekkuURL := flag.String("indiekku-url", "http://localhost:3000", "indiekku API URL")
	flag.Parse()

	if *publicIP == "" {
		fmt.Fprintln(os.Stderr, "error: --public-ip is required")
		flag.Usage()
		os.Exit(1)
	}

	// Load indiekku API key from the standard location
	_, err := security.LoadAPIKey()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: could not load indiekku API key: %v\n", err)
		fmt.Fprintln(os.Stderr, "Make sure 'indiekku serve' has been run at least once.")
		os.Exit(1)
	}

	// Auto-generate token secret if not provided
	secret := *tokenSecret
	if secret == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to generate token secret: %v\n", err)
			os.Exit(1)
		}
		secret = hex.EncodeToString(b)
		fmt.Println("warning: --token-secret not set, using a random secret (join tokens will be invalid after restart)")
	}

	indiekkuClient, err := client.NewClient(*indiekkuURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := indiekkuClient.HealthCheck(); err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot reach indiekku at %s: %v\n", *indiekkuURL, err)
		fmt.Fprintln(os.Stderr, "Make sure 'indiekku serve' is running.")
		os.Exit(1)
	}

	handler := matchmaking.NewHandler(indiekkuClient, *publicIP, secret)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.GET("/health", handler.Health)
	r.GET("/servers", handler.ListServers)
	r.POST("/match", handler.Match)
	r.POST("/join/:name", handler.Join)

	fmt.Printf("✓ indiekku-match started\n")
	fmt.Printf("  Listening: 0.0.0.0:%s\n", *port)
	fmt.Printf("  Public IP: %s\n", *publicIP)
	fmt.Printf("  Max players per server: %d\n", *maxPlayers)
	fmt.Printf("  indiekku: %s\n", *indiekkuURL)

	if err := r.Run(":" + *port); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
