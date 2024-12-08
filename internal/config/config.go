package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds application configuration and credentials
type Config struct {
	APIID             int    // Telegram API ID
	APIHash           string // Telegram API Hash
	Phone             string // Phone number for authentication
	RecipientUsername string // Username to forward messages to
}

// NewConfigFromEnv creates Config from environment variables
func NewConfigFromEnv() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("loading .env: %w", err)
	}

	apiID, err := strconv.Atoi(os.Getenv("API_ID"))
	if err != nil {
		return nil, fmt.Errorf("parsing API_ID: %w", err)
	}

	return &Config{
		APIID:             apiID,
		APIHash:           os.Getenv("API_HASH"),
		Phone:             os.Getenv("PHONE"),
		RecipientUsername: os.Getenv("RECIPIENT_USERNAME"),
	}, nil
}
