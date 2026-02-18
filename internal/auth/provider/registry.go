package provider

import "fmt"

// Registry holds all configured OAuth providers and allows
// lookup by provider name. It performs no auth logic itself.
type Registry struct {
	providers map[string]OAuthProvider
}

// NewRegistry registers the given OAuth providers by name.
// Provider names must be unique.
func NewRegistry(list ...OAuthProvider) *Registry {
	m := make(map[string]OAuthProvider)
	for _, p := range list {
		m[p.Name()] = p
	}
	return &Registry{providers: m}
}

// Get returns the OAuth provider by name or an error if not registered.
func (r *Registry) Get(name string) (OAuthProvider, error) {
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown oauth provider: %s", name)
	}
	return p, nil
}
