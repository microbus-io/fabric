package dlru

type cacheOptions struct {
	Bump bool
}

// LoadOption is used customize loading from the cache.
type LoadOption func(opts *cacheOptions)

// NoBump prevents a loaded element from being bumped to the head of the cache.
func NoBump() LoadOption {
	return func(opts *cacheOptions) {
		opts.Bump = false
	}
}

// Bump causes a loaded element to be bumped to the head of the cache.
// This is the default behavior.
func Bump(bump bool) LoadOption {
	return func(opts *cacheOptions) {
		opts.Bump = bump
	}
}
