package vault

import (
	"errors"
	"os"
)

// APIKey returns the credential used to talk to the sync service, taken from
// the environment so it never lives in source control. Missing configuration
// is an error — a silent fallback credential would defeat the point.
func APIKey() (string, error) {
	key := os.Getenv("VAULT_API_KEY")
	if key == "" {
		return "", errors.New("VAULT_API_KEY is not set")
	}
	return key, nil
}
