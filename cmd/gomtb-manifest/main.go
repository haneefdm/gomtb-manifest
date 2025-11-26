package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/haneefdm/gomtb-manifest/mtbmanifest"
	"github.com/jessevdk/go-flags"
)

type Timer struct {
	startTime int64
}

func NewTimer() *Timer {
	var ret = &Timer{
		startTime: NowMs(),
	}
	return ret
}

func (t *Timer) Start() {
	t.startTime = NowMs()
}

func NowMs() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func (t *Timer) ElapsedMs() int64 {
	return NowMs() - t.startTime
}

var CY_TOOLS_PATH = "/Applications/MoodusToolbox/tools_3.6"
var SuperManifestURL = "https://github.com/Infineon/mtb-super-manifest/raw/v2.X/mtb-super-manifest-fv2.xml"
var ProxyUrl = "" // e.g., "http://user:password@your_proxy_host:your_proxy_port"

var options struct {
	// We should change this to LogLevel or similar later
	Verbose  bool `short:"v" long:"verbose" description:"Enable verbose logging"`
	showHelp bool `short:"h" long:"help" description:"Show help message"`
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("recovered from panic: %v", r)
			os.Exit(1)
		}
	}()
	doMain()
}

func doMain() {
	_, err := flags.Parse(&options)
	if err != nil {
		fmt.Printf("Error parsing command-line options: %v\n", err)
		return
	}
	if options.showHelp {
		flags.NewParser(&options, flags.Default).WriteHelp(os.Stdout)
		return
	}

	timer := NewTimer()
	// For demonstration, we will just ingest the manifest and print the number of boards
	superManifest, err := ingestManifestTree()
	if err != nil {
		fmt.Printf("Error ingesting manifest: %v\n", err)
		return
	}
	fmt.Printf("Finished ingesting super manifest in %d ms\n", timer.ElapsedMs())
	_ = superManifest // To avoid unused variable warning during development
	os.Exit(0)

	jsonData, err := json.MarshalIndent(superManifest, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling super manifest to JSON: %v\n", err)
		return
	}
	fmt.Printf("Ingested info from super manifest:\n%s\n", string(jsonData))
}

func ingestManifestTree() (*mtbmanifest.SuperManifest, error) {
	// Example usage of fetching and reading the super manifest
	fmt.Println("Fetching super manifest...")
	urlFetcher := NewManifestFetcher(10)

	superData, err := urlFetcher.Cache.Get(SuperManifestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch super manifest %s: %v", SuperManifestURL, err)
	}
	superManifest, err := UnmarshalManifest(superData, err, mtbmanifest.ReadSuperManifest)
	if err != nil {
		return nil, fmt.Errorf("failed to parse super manifest %s: %v", SuperManifestURL, err)
	}
	fmt.Printf("Fetched super manifest with %d board manifests\n", len(superManifest.BoardManifestList.BoardManifest))

	urls := []FetchUrlWithCb{}
	var mu sync.Mutex
	for ix, mManifest := range superManifest.BoardManifestList.BoardManifest {
		item := FetchUrlWithCb{
			Url: mManifest.URI, Index: ix,
			Callback: func(urlStr string, data []byte, err error, index int) {
				fmt.Printf("Board: %s: len=%d, err=%v, index=%d\n", urlStr, len(data), err, index)
				board, err := UnmarshalManifest(data, err, mtbmanifest.ReadBoardManifest)
				if err != nil {
					fmt.Printf("Error fetching %s: %v\n", urlStr, err)
				} else {
					mu.Lock()
					superManifest.BoardManifestList.BoardManifest[index].Boards = board
					mu.Unlock()
				}
			},
		}
		urls = append(urls, item)
	}

	for ix, aManifest := range superManifest.AppManifestList.AppManifest {
		item := FetchUrlWithCb{
			Url: aManifest.URI, Index: ix,
			Callback: func(urlStr string, data []byte, err error, index int) {
				fmt.Printf("App: %s: len=%d, err=%v, index=%d\n", urlStr, len(data), err, index)
				app, err := UnmarshalManifest(data, err, mtbmanifest.ReadAppsManifest)
				if err != nil {
					fmt.Printf("Error fetching %s: %v\n", urlStr, err)
				} else {
					mu.Lock()
					superManifest.AppManifestList.AppManifest[index].Apps = app
					mu.Unlock()
				}
			},
		}
		urls = append(urls, item)
	}
	for ix, mManifest := range superManifest.MiddlewareManifestList.MiddlewareManifest {
		item := FetchUrlWithCb{
			Url: mManifest.URI, Index: ix,
			Callback: func(urlStr string, data []byte, err error, index int) {
				fmt.Printf("Middleware: %s: len=%d, err=%v, index=%d\n", urlStr, len(data), err, index)
				middleware, err := UnmarshalManifest(data, err, mtbmanifest.ReadMiddlewareManifest)
				if err != nil {
					fmt.Printf("Error fetching file %s: %v\n", urlStr, err)
				} else {
					mu.Lock()
					superManifest.MiddlewareManifestList.MiddlewareManifest[index].Middlewares = middleware
					mu.Unlock()
				}
			},
		}
		urls = append(urls, item)
	}

	urlFetcher.FetchAllWithCb(urls)
	return superManifest, err
}

func UnmarshalManifest[T any](data []byte, err error, parseFunc func([]byte) (*T, error)) (*T, error) {
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %v", err)
	}
	manifest, err := parseFunc(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %v", err)
	}
	return manifest, nil
}

func UnmarshalXmlManifest[T any](item any, unmarshalFunc func([]byte) (*T, error)) (*T, error) {
	err := item.(error)
	if err != nil {
		return nil, err
	}
	return unmarshalFunc(item.([]byte))
}

func FetchManifest[T any](fileURL string, parseFunc func([]byte) (*T, error)) (*T, error) {
	content, err := GetUrlContent(fileURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest from %s: %v", fileURL, err)
	}
	manifest, err := parseFunc(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse manifest from %s: %v", fileURL, err)
	}
	return manifest, nil
}
