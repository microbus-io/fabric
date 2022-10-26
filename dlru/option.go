package dlru

type cacheOptions struct {
	Bump      bool
	Consensus bool
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
func Bump() LoadOption {
	return func(opts *cacheOptions) {
		opts.Bump = true
	}
}

// Consensus indicates to check with all peers for consistency before returning an
// element from the cache.
// This option impacts performance. It is on by default.
func Consensus() LoadOption {
	return func(opts *cacheOptions) {
		opts.Consensus = true
	}
}

// Quick indicates not to check with peers before returning an element that is found in the local cache.
// This option improves performance. It is off by default.
func Quick() LoadOption {
	return func(opts *cacheOptions) {
		opts.Consensus = false
	}
}
