package auth

// Identity represents a normalized external authentication identity
// returned by an OAuth provider. It contains facts only, no decisions.
type Identity struct {
	Provider       string // e.g. "google", "linkedin"
	ProviderUserID string // provider-scoped unique user identifier (sub)
	Email          string // verified email returned by provider
	EmailVerified  bool   // whether provider asserts email ownership
}
