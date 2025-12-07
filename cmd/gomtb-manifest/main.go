package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
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

type Logger struct {
	Logger *log.Logger
}

var logger = &Logger{
	Logger: log.New(os.Stdout, "", log.LstdFlags),
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.Logger.Printf("[INFO] "+format, args...)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Logger.Printf("[DEBUG] "+format, args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logger.Printf("[ERROR] "+format, args...)
}

func (l *Logger) Warningf(format string, args ...interface{}) {
	l.Logger.Printf("[WARNING] "+format, args...)
}

var CY_TOOLS_PATH = "/Applications/MoodusToolbox/tools_3.6"
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
	mtbmanifest.SetLogger(logger)
	_, err := flags.Parse(&options)
	if err != nil {
		logger.Errorf("Error parsing command-line options: %v\n", err)
		return
	}
	if options.showHelp {
		flags.NewParser(&options, flags.Default).WriteHelp(os.Stdout)
		return
	}

	mtbmanifest.EnableXMLUnmarshalVerification(true)

	timer := NewTimer()
	// For demonstration, we will just ingest the manifest and print the number of boards
	superManifest, err := mtbmanifest.NewSuperManifestFromURL("")
	if err != nil {
		logger.Errorf("Error ingesting manifest: %v\n", err)
		return
	}

	logger.Infof("Finished ingesting super manifest in %d ms\n", timer.ElapsedMs())

	name := "KIT_PSE84_EVAL_EPC2"
	board := (*superManifest.GetBoardsMap())[name]
	if board != nil {
		logger.Infof("Found board %s:\n", name)
		jsonData, _ := json.MarshalIndent(board, "", "  ")
		_ = os.WriteFile("tmp/board.json", jsonData, 0644)
		jsonData, _ = json.MarshalIndent(board.Capabilities, "", "  ")
		_ = os.WriteFile("tmp/capabilities.json", jsonData, 0644)
	} else {
		logger.Errorf("Error: Board %s not found\n", name)
	}
	if true {
		jsonData, _ := json.MarshalIndent(superManifest.GetMiddlewareMap(), "", "  ")
		_ = os.WriteFile("tmp/middleware.json", jsonData, 0644)
		mwItems := mtbmanifest.FindMiddlewareForBoard(superManifest, board)
		logger.Infof("Middleware matched for board %s: %d items\n", name, len(mwItems))
		mwMapByCategory := make(map[string][]*mtbmanifest.MiddlewareItem)
		for _, mw := range mwItems {
			mwMapByCategory[mw.Category] = append(mwMapByCategory[mw.Category], mw)
		}
		for category, items := range mwMapByCategory {
			fmt.Printf("Category: %s\n", category)
			for _, mw := range items {
				fmt.Printf("    %s: %s\n", mw.ID, mw.URI)
			}
		}
	}
	os.Exit(0)
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
