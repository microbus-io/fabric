package dlru

type cacheOptions struct {
	Bump      bool
	PeerCheck bool
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

// PeerCheck indicates to check with all peers for consistency before returning an
// element from the cache.
// This option impacts performance. It is on by default.
func PeerCheck(check bool) LoadOption {
	return func(opts *cacheOptions) {
		opts.PeerCheck = check
	}
}

// NoPeerCheck indicates not to check with peers before returning an element that is found in the local cache.
// This option improves performance. It is off by default.
func NoPeerCheck() LoadOption {
	return func(opts *cacheOptions) {
		opts.PeerCheck = false
	}
}
