package mtbmanifest

import "time"

// Examples demonstrating the functional options pattern for ManifestFetcher
//
// This file shows how clients can configure ManifestFetcher using the
// functional options pattern, which provides flexibility while maintaining
// sensible defaults.

// Example 1: Use default settings
// Creates a fetcher with default cache and conservative concurrency (10)
func ExampleNewManifestFetcher_defaults() {
	fetcher := NewManifestFetcher()
	_ = fetcher // Use the fetcher...
}

// Example 2: Customize only concurrency
// Uses default cache but allows more concurrent fetches
func ExampleNewManifestFetcher_customConcurrency() {
	fetcher := NewManifestFetcher(WithMaxConcurrent(20))
	_ = fetcher // Use the fetcher...
}

// Example 3: Customize cache location and TTL
// Perfect for testing or when you need a specific cache location
func ExampleNewManifestFetcher_customCache() {
	// Create a custom cache in a specific location with 7-day TTL
	customCache := NewManifestCache("/tmp/my-test-cache", 7*24*time.Hour)

	fetcher := NewManifestFetcher(WithCache(customCache))
	_ = fetcher // Use the fetcher...
}

// Example 4: Customize both cache and concurrency
// Full control over both aspects
func ExampleNewManifestFetcher_customBoth() {
	// Create a custom cache
	customCache := NewManifestCache("/var/cache/manifests", 30*24*time.Hour)

	// Create fetcher with both custom cache and high concurrency
	fetcher := NewManifestFetcher(
		WithCache(customCache),
		WithMaxConcurrent(50),
	)
	_ = fetcher // Use the fetcher...
}

// Example 5: Proper cleanup with defer
// Always use defer to ensure graceful shutdown
func ExampleManifestFetcher_properCleanup() {
	fetcher := NewManifestFetcher()
	defer fetcher.Cache().Close() // Ensures background worker stops gracefully

	// Use the fetcher...
	data, err := fetcher.Cache().Get("https://example.com/manifest.xml")
	if err != nil {
		// Handle error...
	}
	_ = data

	// When function exits, Close() is called automatically
	// Safe to call multiple times - it's idempotent
}

// Example 6: Accessing the cache
// The Cache() method provides read-only access to the underlying cache
func ExampleManifestFetcher_accessCache() {
	fetcher := NewManifestFetcher()
	defer fetcher.Cache().Close()

	// Get data through the fetcher
	data, err := fetcher.Cache().Get("https://example.com/manifest.xml")
	if err != nil {
		// Handle error...
	}
	_ = data

	// Clear stale entries
	_ = fetcher.Cache().ClearStale()
} // Why use functional options?
//
// Benefits:
// 1. **Backward Compatible**: Adding new options doesn't break existing code
// 2. **Clear Intent**: Each option is self-documenting (WithCache, WithMaxConcurrent)
// 3. **Sensible Defaults**: Works out-of-the-box without configuration
// 4. **Flexible**: Clients configure only what they need
// 5. **Composable**: Options can be stored, combined, and reused
//
// Compare to alternatives:
//
// Config struct approach (less flexible):
//   type FetcherConfig struct {
//       Cache *ManifestCache
//       MaxConcurrent int
//   }
//   func NewManifestFetcher(cfg FetcherConfig) // Must always provide config
//
// Multiple constructors (doesn't scale):
//   func NewManifestFetcher()
//   func NewManifestFetcherWithCache(cache *ManifestCache)
//   func NewManifestFetcherWithConcurrency(max int)
//   func NewManifestFetcherWithBoth(cache *ManifestCache, max int)
//
// The functional options pattern is the idiomatic Go way to solve this problem.
