package mtbmanifest

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
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
// How it works:

// 1. First request (cold start, no cache):
//    User request → Cache miss → Fetch from network (2-4s) → Return data

// 2. Subsequent requests (cache fresh, age < TTL):
//    User request → Cache hit → Return immediately (2ms) → Done

// 3. Subsequent requests (cache stale, age >= TTL):
//    User request → Cache hit → Return stale data (2ms) → Queue background refresh
//    Background → Fetch fresh data → Update cache → Ready for next request

// 4. Network failure (background refresh fails):
//    User request → Cache hit → Return stale data (better than error!)
//    Background → Fetch fails → Log error → Keep using stale data
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

const (
	compressionThreshold = 10 * 1024 // 10KB
	compressionFlag      = 0x01
	defaultTTL           = 15 * 24 * time.Hour // 15 days
)

func NewManifestCache(cacheDir string, ttl time.Duration) *ManifestCache {
	if cacheDir == "" {
		home, _ := os.UserHomeDir()
		cacheDir = filepath.Join(home, ".modustoolbox", "mtbmcp", "manifests")
	}
	if ttl <= 0 {
		ttl = defaultTTL
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

func NewManifestDefaultCache() *ManifestCache {
	return NewManifestCache("", 0)
}

// Call this when your program is shutting down
func (c *ManifestCache) Close() {
	close(c.refreshQueue)
}

func (c *ManifestCache) Get(urlStr string) ([]byte, error) {
	data, err := c.readCache(urlStr)
	if err == nil {
		// Cache hit - check if stale
		info, _ := os.Stat(c.urlToFilename(urlStr))
		age := time.Since(info.ModTime())

		if age >= c.ttl {
			// Stale - queue for background refresh
			c.queueRefresh(urlStr)
		}

		// Return cached data immediately (stale or not)
		return data, nil
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

	err = c.writeCache(urlStr, data)
	if err != nil {
		log.Printf("Warning: failed to write cache for %s: %v", urlStr, err)
	}
	return data, nil
}

func (c *ManifestCache) fetchFromNetwork(urlStr string) ([]byte, error) {
	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

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
			oldUrl, err := c.readUrlFromCache(filepath.Join(c.cacheDir, entry.Name()))
			if err == nil && oldUrl != "" {
				c.queueRefresh(oldUrl)
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
	var wgFetches sync.WaitGroup
	var wgCallbacks sync.WaitGroup
	// errChan := make(chan error, len(urls))

	for ix, item := range urls {
		wgFetches.Add(1)
		go func(index int, item FetchUrlWithCb) {
			defer wgFetches.Done()

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
				wgCallbacks.Add(1)
				go func(url string, data []byte, err error, index int) {
					item.Callback(url, data, err, index)
					wgCallbacks.Done()
				}(item.Url, data, err, item.Index)
			}
		}(ix, item)
	}

	wgFetches.Wait()
	wgCallbacks.Wait()
	// close(errChan)
	return results
}

// The return value is a map of URL to fetched data or any error encountered
func (f *ManifestFetcher) FetchAll(urls []string) map[string]any {
	results := map[string]any{}
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, urlStr := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()

			f.limiter <- struct{}{}        // Acquire
			defer func() { <-f.limiter }() // Release

			data, err := f.Cache.Get(u)

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
	return results
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
			_ = os.Remove(filepath.Join(c.cacheDir, entry.Name()))
		}
	}
	return nil
}

// Cache file header structure. DO NOT CHANGE!
// If you need to change, bump the version number and handle old versions in code.
// One simple way would be to invalidate old versions. But version HAS to be the 3rd byte.
// Also, the magic number has to be the first two bytes and changing that would also invalidate old caches.
type CacheHeader struct {
	Magic    [2]byte
	Version  uint8
	Flags    uint8 // bit 0: compressed
	Checksum uint8 // simple checksum of URL bytes
	URLSize  uint16
}

func validateHeader(header *CacheHeader, urlStr string) error {
	if header.Magic != [2]byte{'M', 'C'} {
		return fmt.Errorf("invalid magic number")
	}
	if header.Version != 1 {
		return fmt.Errorf("unsupported version %d", header.Version)
	}
	urlBytes := []byte(urlStr)
	if header.Checksum != simpleChecksum(urlBytes) {
		return fmt.Errorf("checksum mismatch")
	}
	return nil
}

func (c *ManifestCache) writeCache(urlStr string, content []byte) error {
	err := os.MkdirAll(c.cacheDir, 0o755)
	if err != nil {
		return err
	}
	filename := c.urlToFilename(urlStr)
	urlBytes := []byte(urlStr)

	// Decide: compress or not?
	shouldCompress := len(content) > compressionThreshold

	var finalContent []byte
	var flags uint8

	if shouldCompress {
		// Compress with gzip (stdlib, widely compatible)
		var buf bytes.Buffer
		gzw := gzip.NewWriter(&buf)
		_, _ = gzw.Write(content)
		_ = gzw.Close()

		compressed := buf.Bytes()

		// Only use compression if it actually helped
		if len(compressed) < len(content) {
			finalContent = compressed
			flags |= compressionFlag
		} else {
			finalContent = content
			flags = 0
		}
	} else {
		finalContent = content
		flags = 0
	}

	// Build header
	header := CacheHeader{
		Magic:    [2]byte{'M', 'C'},
		Version:  1,
		Flags:    flags,
		Checksum: simpleChecksum(urlBytes),
		URLSize:  uint16(len(urlBytes)),
	}

	// Write atomically to temp file, then rename
	tmpFile := filename + ".tmp"
	f, err := os.Create(tmpFile)
	if err != nil {
		return err
	}
	closed := false
	defer func() {
		if !closed {
			_ = f.Close()
		}
	}()

	err = binary.Write(f, binary.BigEndian, &header)
	if err != nil {
		return err
	}
	_, err = f.Write(urlBytes)
	if err != nil {
		return err
	}
	_, err = f.Write(finalContent)
	if err != nil {
		return err
	}
	closed = true
	_ = f.Close() // We have a defer close above. But needs to be closed before rename

	// Atomic rename (even on Windows)
	return os.Rename(tmpFile, filename)
}

func (c *ManifestCache) readCache(urlStr string) ([]byte, error) {
	filename := c.urlToFilename(urlStr)
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	// Read and validate header
	var header CacheHeader
	if err := binary.Read(f, binary.BigEndian, &header); err != nil {
		return nil, err
	}

	// Read URL and validate
	urlBytes := make([]byte, header.URLSize)
	_, err = io.ReadFull(f, urlBytes)
	if err != nil {
		return nil, err
	}
	readUrlStr := string(urlBytes)
	if readUrlStr != urlStr {
		return nil, fmt.Errorf("URL mismatch in cache")
	}
	if err := validateHeader(&header, readUrlStr); err != nil {
		return nil, err
	}

	// Read content
	content, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	// Decompress if needed
	if header.Flags&compressionFlag != 0 {
		gzr, err := gzip.NewReader(bytes.NewReader(content))
		if err != nil {
			return nil, err
		}
		_ = gzr.Close()
		return io.ReadAll(gzr)
	}

	return content, nil
}

func (c *ManifestCache) readUrlFromCache(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	// Read and validate header
	var header CacheHeader
	if err := binary.Read(f, binary.BigEndian, &header); err != nil {
		return "", err
	}

	// Read URL and validate
	urlBytes := make([]byte, header.URLSize)
	_, err = io.ReadFull(f, urlBytes)
	if err != nil {
		return "", err
	}
	urlStr := string(urlBytes)
	if err := validateHeader(&header, urlStr); err != nil {
		return "", err
	}
	return urlStr, nil
}

func simpleChecksum(data []byte) uint8 {
	var sum uint8
	for _, b := range data {
		sum ^= b
	}
	return sum
}
