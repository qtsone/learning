package vault

import (
	"strings"
	"testing"
)

func TestAPIKeyComesFromEnvironment(t *testing.T) {
	t.Setenv("VAULT_API_KEY", "from-env-123")
	got, err := APIKey()
	if err != nil {
		t.Fatalf("APIKey() error with VAULT_API_KEY set: %v", err)
	}
	if got != "from-env-123" {
		t.Errorf("APIKey() = %q, want %q — the key must come from the environment, not the source code", got, "from-env-123")
	}
	if strings.HasPrefix(got, "sk_live_") {
		t.Errorf("APIKey() still returns the hardcoded credential")
	}
}

func TestAPIKeyErrorsWhenUnset(t *testing.T) {
	t.Setenv("VAULT_API_KEY", "")
	if got, err := APIKey(); err == nil {
		t.Errorf("APIKey() = %q, nil — want an error when VAULT_API_KEY is empty, never a baked-in fallback", got)
	}
}
