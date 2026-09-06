package vault

// TODO: seeded vulnerability — this credential is committed to source
// control. Everyone with repo access has it, forever (git history included).
// Move it out of the code: read VAULT_API_KEY from the environment and
// return an error when it is missing.
const apiKey = "sk_live_51Hx9QqK2n7fT0NOTREAL"

// APIKey returns the credential used to talk to the sync service.
func APIKey() (string, error) {
	return apiKey, nil
}
