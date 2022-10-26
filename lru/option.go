package lru

// --- Load options ---

type loadOptions interface {
	SetBump(bump bool)
}

// LoadOption is used customize loading from the cache.
type LoadOption func(opts loadOptions)

// Bump indicates whether or not a loaded element is bumped to the head of the cache.
// This is on by default.
func Bump(bump bool) LoadOption {
	return func(opts loadOptions) {
		opts.SetBump(bump)
	}
}

// NoBump indicates not to bump a loaded element to the head of the cache.
func NoBump() LoadOption {
	return Bump(false)
}

// --- Store options ---

type storeOptions interface {
	SetWeight(weight int)
}

// StoreOption is used customize storing in the cache.
type StoreOption func(opts storeOptions)

// Weight sets the weight of the element stored in the cache.
// It must be greater than 0 and cannot exceed the cache's maximum weight limit.
// Elements are evicted when the total weight exceeds the cache's limit.
func Weight(weight int) StoreOption {
	return func(opts storeOptions) {
		if weight > 0 {
			opts.SetWeight(weight)
		}
	}
}

// --- LoadOrStore options ---

type loadOrStoreOptions interface {
	loadOptions
	storeOptions
}

// LoadOrStoreOption is used customize the load or store operation.
type LoadOrStoreOption func(opts loadOrStoreOptions)

// --- Implementation ---

type cacheOptions struct {
	Bump   bool
	Weight int
}

// SetBump sets whether a loaded element is bumped to the head of the cache.
func (co *cacheOptions) SetBump(bump bool) {
	co.Bump = bump
}

// SetWeight sets the weight of a stored element.
func (co *cacheOptions) SetWeight(weight int) {
	co.Weight = weight
}
