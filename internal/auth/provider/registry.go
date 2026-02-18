package provider

type Registry struct {
	provider OAuthProvider
}

// NewRegistry registers a single OAuth provider.
func NewRegistry(p OAuthProvider) *Registry {
	return &Registry{provider: p}
}

// Get returns the configured OAuth provider.
func (r *Registry) Get() OAuthProvider {
	return r.provider
}
