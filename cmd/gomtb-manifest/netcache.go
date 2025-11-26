package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

/*
```

**How it works:**

1. **First request** (cold start, no cache):
```

	User request → Cache miss → Fetch from network (2-4s) → Return data

```
2. **Subsequent requests** (cache fresh, age < TTL):
```

	User request → Cache hit → Return immediately (2ms) → Done

```
3. **Subsequent requests** (cache stale, age >= TTL):
```

	User request → Cache hit → Return stale data (2ms) → Queue background refresh
	Background → Fetch fresh data → Update cache → Ready for next request

```
4. **Network failure** (background refresh fails):
```

	User request → Cache hit → Return stale data (better than error!)
	Background → Fetch fails → Log error → Keep using stale data
*/

type ManifestFetcher struct {
	Cache   *ManifestCache
	limiter chan struct{} // Rate limit concurrent fetches
}

type ManifestCache struct {
	cacheDir string
	ttl      time.Duration

	// Background refresh tracking
	refreshQueue chan string
	refreshing   sync.Map // track URLs being refreshed
}

const defaultTTL = 15 * 24 * time.Hour // 15 days

func NewManifestCache(cacheDir string, ttl time.Duration) *ManifestCache {
	if cacheDir == "" {
		home, _ := os.UserHomeDir()
		cacheDir = filepath.Join(home, ".modustoolbox", "mtbmcp", "manifests")
	}

	c := &ManifestCache{
		cacheDir:     cacheDir,
		ttl:          ttl,
		refreshQueue: make(chan string, 100),
	}

	// Start background refresh worker
	go c.refreshWorker()

	return c
}

// Call this when your program is shutting down
func (c *ManifestCache) Close() {
	close(c.refreshQueue)
}

func (c *ManifestCache) Get(urlStr string) ([]byte, error) {
	cacheFile := c.urlToFilename(urlStr)

	// Try to read from cache first (even if stale)
	data, err := os.ReadFile(cacheFile)
	if err == nil {
		_, data, err = decodeBytesToUrl(data)
		if err == nil {
			// Got cached data - check if stale
			info, _ := os.Stat(cacheFile)
			age := time.Since(info.ModTime())

			if age >= c.ttl {
				// Stale - queue for background refresh
				c.queueRefresh(urlStr)
			}

			// Return cached data immediately (stale or not)
			return data, nil
		}
	}

	// Cache miss - must fetch synchronously
	return c.fetchAndCache(urlStr)
}

func (c *ManifestCache) queueRefresh(urlStr string) {
	// Avoid duplicate refreshes
	if _, alreadyQueued := c.refreshing.LoadOrStore(urlStr, true); alreadyQueued {
		return
	}

	select {
	case c.refreshQueue <- urlStr:
		// Queued successfully
	default:
		// Queue full - skip this refresh
		c.refreshing.Delete(urlStr)
	}
}

func (c *ManifestCache) refreshWorker() {
	// Process refresh queue in background
	for urlStr := range c.refreshQueue {
		// Refresh this URL
		_, err := c.fetchAndCache(urlStr)
		if err != nil {
			log.Printf("Background refresh failed for %s: %v", urlStr, err)
		}

		// Mark as no longer refreshing
		c.refreshing.Delete(urlStr)

		// Small delay to avoid hammering servers
		time.Sleep(100 * time.Millisecond)
	}
}

func (c *ManifestCache) fetchAndCache(urlStr string) ([]byte, error) {
	data, err := c.fetchFromNetwork(urlStr)
	if err != nil {
		return nil, err
	}

	// Save to cache
	cacheFile := c.urlToFilename(urlStr)
	os.MkdirAll(filepath.Dir(cacheFile), 0755)
	encodedData := encodeUrlToBytes(urlStr)
	encodedData = append(encodedData, data...)
	if err := os.WriteFile(cacheFile, encodedData, 0644); err != nil {
		log.Printf("Warning: failed to cache %s: %v", urlStr, err)
	}

	return data, nil
}

func (c *ManifestCache) fetchFromNetwork(urlStr string) ([]byte, error) {
	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (c *ManifestCache) urlToFilename(urlStr string) string {
	parsed, _ := url.Parse(urlStr)
	name := parsed.Host + parsed.Path
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, "?", "_")
	return filepath.Join(c.cacheDir, name)
}

func (c *ManifestCache) RefreshAllStale() {
	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, _ := entry.Info()
		if time.Since(info.ModTime()) >= c.ttl {
			data, err := os.ReadFile(filepath.Join(c.cacheDir, entry.Name()))
			if err == nil {
				oldUrl, _, err := decodeBytesToUrl(data)
				if err == nil {
					tmp := c.urlToFilename(oldUrl)
					if tmp == entry.Name() {
						c.queueRefresh(oldUrl)
					}
				}
			}
		}
	}
}

func NewManifestFetcher(maxConcurrent int) *ManifestFetcher {
	return &ManifestFetcher{
		Cache:   NewManifestCache("", defaultTTL),
		limiter: make(chan struct{}, maxConcurrent), // e.g., 10
	}
}

type FetchUrlWithCb struct {
	Url      string
	Index    int
	Callback func(urlString string, data []byte, err error, index int)
}

// The return value is a map of URL to fetched data or any error encountered
// If a Callback is provided, it is called for each URL when fetched and it will be
// in its own goroutine. So, use callbacks with proper synchronization if needed.
// The order of the callbacks can be different from the order of the input URLs.
func (f *ManifestFetcher) FetchAllWithCb(urls []FetchUrlWithCb) map[string]any {
	results := map[string]any{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	// errChan := make(chan error, len(urls))

	for ix, item := range urls {
		wg.Add(1)
		go func(index int, item FetchUrlWithCb) {
			defer wg.Done()

			f.limiter <- struct{}{}        // Acquire
			defer func() { <-f.limiter }() // Release

			data, err := f.Cache.Get(item.Url)
			// if err != nil {
			// 	errChan <- fmt.Errorf("%s: %w", u, err)
			// 	return
			// }

			mu.Lock()
			if err != nil {
				results[item.Url] = err
			} else {
				results[item.Url] = data
			}
			mu.Unlock()
			if item.Callback != nil {
				item.Callback(item.Url, data, err, item.Index)
			}
		}(ix, item)
	}

	wg.Wait()
	// close(errChan)
	return results
}

// The return value is a map of URL to fetched data or any error encountered
func (f *ManifestFetcher) FetchAll(urls []string) map[string]any {
	results := map[string]any{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	// errChan := make(chan error, len(urls))

	for _, urlStr := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()

			f.limiter <- struct{}{}        // Acquire
			defer func() { <-f.limiter }() // Release

			data, err := f.Cache.Get(u)
			// if err != nil {
			// 	errChan <- fmt.Errorf("%s: %w", u, err)
			// 	return
			// }

			mu.Lock()
			if err != nil {
				results[u] = err
			} else {
				results[u] = data
			}
			mu.Unlock()
		}(urlStr)
	}

	wg.Wait()
	// close(errChan)
	return results

	/*
		// Collect any errors
		var errs []error
		for err := range errChan {
			errs = append(errs, err)
		}
		if len(errs) > 0 {
			return results, fmt.Errorf("fetch errors: %v", errs)
		}

		return results, nil
	*/
}

// Add to cache struct
func (c *ManifestCache) Clear() error {
	return os.RemoveAll(c.cacheDir)
}

func (c *ManifestCache) ClearStale() error {
	entries, _ := os.ReadDir(c.cacheDir)
	for _, entry := range entries {
		info, _ := entry.Info()
		if time.Since(info.ModTime()) > c.ttl {
			os.Remove(filepath.Join(c.cacheDir, entry.Name()))
		}
	}
	return nil
}

const magic uint16 = 0x4D43 // 'MC' for "Manifest Cache"

// Helper functions to encode/decode URL with length prefix
func encodeUrlToBytes(urlStr string) []byte {
	tmp := []byte(urlStr)
	count := len(tmp)
	ret := make([]byte, 4+count)
	ret[0] = byte(magic >> 8)
	ret[1] = byte(magic & 0xff)
	ret[2] = byte(count >> 8)
	ret[3] = byte(count & 0xff)
	copy(ret[4:], tmp)
	return ret
}

// Returns decoded URL, remaining bytes, error
func decodeBytesToUrl(data []byte) (string, []byte, error) {
	if len(data) < 4 {
		return "", data, fmt.Errorf("data too short to decode URL")
	}
	magicRead := uint16(data[0])<<8 | uint16(data[1])
	if magicRead != magic {
		return "", data, fmt.Errorf("invalid magic number")
	}
	count := int(data[2])<<8 | int(data[3])
	if len(data) < 4+count {
		return "", data, fmt.Errorf("data length mismatch")
	}
	return string(data[4 : 4+count]), data[4+count:], nil
}
