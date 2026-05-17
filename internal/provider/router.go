package provider

import "fmt"

// Router maps model names to their corresponding Provider implementations.
type Router struct {
	providers map[string]Provider
}

// NewRouter creates a new Router instance.
func NewRouter() *Router {
	return &Router{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the router. If a provider with the same name
// already exists, it is overwritten (last write wins).
func (r *Router) Register(p Provider) {
	r.providers[p.Name()] = p
}

// Resolve finds the provider that supports the given model.
// Returns an error listing all available models if no provider matches.
func (r *Router) Resolve(model string) (Provider, error) {
	for _, p := range r.providers {
		for _, m := range p.SupportedModels() {
			if m == model {
				return p, nil
			}
		}
	}

	return nil, fmt.Errorf("unknown model %q; available models: %v", model, r.AllModels())
}

// AllModels returns all model names from all registered providers.
func (r *Router) AllModels() []string {
	var models []string
	for _, p := range r.providers {
		models = append(models, p.SupportedModels()...)
	}

	return models
}
