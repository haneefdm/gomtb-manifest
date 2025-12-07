package mtbmanifest

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
)

const SuperManifestURL = "https://github.com/Infineon/mtb-super-manifest/raw/v2.X/mtb-super-manifest-fv2.xml"

type LoggerIF interface {
	Infof(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Warningf(format string, args ...interface{})
}

type Logger struct {
	Logger *log.Logger
}

func SetLogger(l LoggerIF) {
	logger = l
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

var defLogger = &Logger{
	Logger: log.New(os.Stdout, "", log.LstdFlags),
}
var logger LoggerIF = defLogger

// SuperManifestIF defines the interface for accessing and managing super manifest data.
// This interface provides methods to retrieve boards, apps, middleware, BSP dependencies,
// BSP capabilities, and merge multiple super manifests.
type SuperManifestIF interface {
	// GetBoardsMap returns a map of all boards indexed by their ID
	GetBoardsMap() *map[string]*Board

	// Get list of board IDs. Order is according to manifest listing.
	GetBoardIDs() []string

	// GetBoard retrieves a specific board by its ID
	GetBoard(boardID string) (*Board, bool)

	// GetAppsMap returns a map of all apps indexed by their ID
	GetAppsMap() *map[string]*App

	// Get list of app IDs. Order is according to manifest listing.
	GetAppIDs() []string

	// GetApp retrieves a specific app by its ID
	GetApp(appID string) (*App, bool)

	// GetMiddlewareMap returns a map of all middleware items indexed by their ID
	GetMiddlewareMap() *map[string]*MiddlewareItem

	// Get list of middleware IDs. Order is according to manifest listing.
	GetMiddlewareIDs() []string

	// GetMiddleware retrieves a specific middleware item by its ID
	GetMiddleware(middlewareID string) (*MiddlewareItem, bool)

	// GetDependencies fetches and caches the BSP dependencies manifest from the given URL
	GetDependencies(urlStr string) *Dependencies

	// GetBSPCapabilitiesManifest fetches and caches the BSP capabilities manifest from the given URL
	GetBSPCapabilitiesManifest(urlStr string) *BSPCapabilitiesManifest

	// GetDependencies retrieves the BSP dependencies for a specific BSP ID from the given URL
	GetDependenciesByID(urlStr string, bspId string) *Depender

	// AddSuperManifestFromURL fetches a super manifest from a URL and merges it into this one
	AddSuperManifestFromURL(urlStr string) error
}

// Super Manifest structures
// This is the root manifest that points to all other manifests. In the future, perhaps
// we should have a data structure above this to manage multiple super manifests? and have
// this one just represent a single super manifest. All the maps would then move up a level.
type SuperManifest struct {
	XMLName                xml.Name                `xml:"super-manifest"`
	Version                string                  `xml:"version,attr"`
	BoardManifestList      *BoardManifestList      `xml:"board-manifest-list"`
	AppManifestList        *AppManifestList        `xml:"app-manifest-list"`
	MiddlewareManifestList *MiddlewareManifestList `xml:"middleware-manifest-list"`

	SourceUrls []string `xml:"-"`

	// Following maps are built on demand for quick lookup from their respective lists
	boardsMap     map[string]*Board
	appMap        map[string]*App
	middlewareMap map[string]*MiddlewareItem

	// Following stores downloaded BSP manifests to avoid re-fetching across multiple boards and manifests
	bspCapabilitiesMap map[string]*BSPCapabilitiesManifest
	dependenciesMap    map[string]*Dependencies

	// Capture unknown tags and attributes
	Surprises []AnyTag   `xml:",any"`
	LostAttrs []xml.Attr `xml:",any,attr"`
}

// NewSuperManifest creates an empty SuperManifest ready to be populated.
func NewSuperManifest() SuperManifestIF {
	ret := &SuperManifest{
		BoardManifestList:      &BoardManifestList{},
		AppManifestList:        &AppManifestList{},
		MiddlewareManifestList: &MiddlewareManifestList{},
		bspCapabilitiesMap:     make(map[string]*BSPCapabilitiesManifest),
		dependenciesMap:        make(map[string]*Dependencies),
	}
	ret.clearMaps()
	return ret
}

// NewSuperManifestFromURL fetches and ingests a complete super manifest tree from the given URL.
// If urlStr is empty, it uses the default SuperManifestURL.
// This constructor fetches all board, app, and middleware manifests concurrently.
func NewSuperManifestFromURL(urlStr string) (SuperManifestIF, error) {
	urlFetcher := NewManifestFetcher(WithMaxConcurrent(runtime.NumCPU()))
	if urlStr == "" {
		urlStr = SuperManifestURL
	}

	// logger.Infof("Fetching super manifest...%s\n", urlStr)
	superData, err := urlFetcher.Cache().Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch super manifest %s: %v", urlStr, err)
	}
	superManifest, err := UnmarshalManifest(superData, err, ReadSuperManifest)
	if err != nil {
		return nil, fmt.Errorf("failed to parse super manifest %s: %v", urlStr, err)
	}
	superManifest.SourceUrls = append(superManifest.SourceUrls, urlStr)
	superManifest.clearMaps()

	urls := []*FetchUrlWithCb{}
	var mu sync.Mutex
	depUrls := make(map[string]interface{})
	capUrls := make(map[string]interface{})
	for ix, mManifest := range superManifest.BoardManifestList.BoardManifest {
		item := &FetchUrlWithCb{
			Url: mManifest.URI, Index: ix,
			Callback: func(urlStr string, data []byte, err error, index int) {
				// logger.Infof("Board: %s: len=%d, err=%v, index=%d\n", urlStr, len(data), err, index)
				boards, err := UnmarshalManifest(data, err, ReadBoardManifest)
				if err != nil {
					logger.Errorf("Error fetching %s: %v\n", urlStr, err)
				} else {
					mu.Lock()
					bm := superManifest.BoardManifestList.BoardManifest[index]
					bm.Boards = boards
					for _, board := range bm.Boards.Boards {
						board.Origin = bm
					}
					mu.Unlock()
				}
			},
		}
		if mManifest.CapabilityURL != "" {
			capUrls[mManifest.CapabilityURL] = mManifest
		}
		if mManifest.DependencyURL != "" {
			depUrls[mManifest.DependencyURL] = mManifest
		}
		urls = append(urls, item)
	}

	for ix, aManifest := range superManifest.AppManifestList.AppManifest {
		item := &FetchUrlWithCb{
			Url: aManifest.URI, Index: ix,
			Callback: func(urlStr string, data []byte, err error, index int) {
				// logger.Infof("App: %s: len=%d, err=%v, index=%d\n", urlStr, len(data), err, index)
				app, err := UnmarshalManifest(data, err, ReadAppsManifest)
				if err != nil {
					logger.Errorf("Error fetching %s: %v\n", urlStr, err)
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
		item := &FetchUrlWithCb{
			Url: mManifest.URI, Index: ix,
			Callback: func(urlStr string, data []byte, err error, index int) {
				// logger.Infof("Middleware: %s: len=%d, err=%v, index=%d\n", urlStr, len(data), err, index)
				middleware, err := UnmarshalManifest(data, err, ReadMiddlewareManifest)
				if err != nil {
					logger.Errorf("Error fetching file %s: %v\n", urlStr, err)
				} else {
					mu.Lock()
					mwM := superManifest.MiddlewareManifestList.MiddlewareManifest[index]
					mwM.Middlewares = middleware
					for _, mw := range mwM.Middlewares.Middlewares {
						mw.Origin = mwM
					}
					mu.Unlock()
				}
			},
		}
		if mManifest.DependencyURL != "" {
			depUrls[mManifest.DependencyURL] = mManifest
		}
		urls = append(urls, item)
	}
	depMap := make(map[string]*Dependencies)
	for depUrl := range depUrls {
		item := &FetchUrlWithCb{
			Url: depUrl,
			Callback: func(urlStr string, data []byte, err error, index int) {
				// logger.Infof("Dependencies: %s: len=%d, err=%v\n", urlStr, len(data), err)
				deps, err := UnmarshalManifest(data, err, ReadDependenciesManifest)
				if err != nil {
					logger.Errorf("Error fetching dependencies %s: %v\n", urlStr, err)
				} else {
					mu.Lock()
					depMap[urlStr] = deps
					mu.Unlock()
				}
			},
		}
		urls = append(urls, item)
	}
	capMap := make(map[string]*BSPCapabilitiesManifest)
	for capUrl := range capUrls {
		item := &FetchUrlWithCb{
			Url: capUrl,
			Callback: func(urlStr string, data []byte, err error, index int) {
				// logger.Infof("Capabilities: %s: len=%d, err=%v\n", urlStr, len(data), err)
				caps, err := UnmarshalManifest(data, err, ReadBSPCapabilitiesManifest)
				if err != nil {
					logger.Errorf("Error fetching capabilities %s: %v\n", urlStr, err)
				} else {
					mu.Lock()
					capMap[urlStr] = caps
					mu.Unlock()
				}
			},
		}
		urls = append(urls, item)
	}

	urlFetcher.FetchAllWithCb(urls)
	superManifest.dependenciesMap = depMap
	superManifest.bspCapabilitiesMap = capMap

	for _, dep := range depMap {
		_ = dep.CreateMaps()
	}
	for _, cap := range capMap {
		_ = cap
		// cap.CreateMaps()
	}

	for depUrl, manifest := range depUrls {
		if boardM, ok := manifest.(*BoardManifest); ok {
			for _, board := range boardM.Boards.Boards {
				if (board.Origin != manifest) || (board.Origin.DependencyURL != depUrl) {
					fmt.Printf("Warning: Board %s origin manifest mismatch for dependency URL %s\n", board.ID, depUrl)
				}
				board.Dependencies = depMap[depUrl].CreateMaps()[board.ID]
			}
		} else if mwM, ok := manifest.(*MiddlewareManifest); ok {
			for _, mw := range mwM.Middlewares.Middlewares {
				if (mw.Origin != manifest) || (mw.Origin.DependencyURL != depUrl) {
					fmt.Printf("Warning: Middleware %s origin manifest mismatch for dependency URL %s\n", mw.ID, depUrl)
				}
				mw.Dependencies = depMap[depUrl].CreateMaps()[mw.ID]
			}
		}
	}
	for capUrl, manifest := range capUrls {
		if boardM, ok := manifest.(*BoardManifest); ok {
			for _, board := range boardM.Boards.Boards {
				if (board.Origin != manifest) || (board.Origin.CapabilityURL != capUrl) {
					fmt.Printf("Warning: Board %s origin manifest mismatch for capability URL %s\n", board.ID, capUrl)
				}
				board.Capabilities = capMap[capUrl]
			}
		}
	}

	logger.Infof("Fetched super manifest with %d boards, %d apps, %d middleware\n",
		len(superManifest.BoardManifestList.BoardManifest),
		len(superManifest.AppManifestList.AppManifest),
		len(superManifest.MiddlewareManifestList.MiddlewareManifest))
	return superManifest, err
}

// Maps are cleared when manifests are merged or modified so that they can be rebuilt on demand
func (sm *SuperManifest) clearMaps() {
	sm.boardsMap = make(map[string]*Board)
	sm.appMap = make(map[string]*App)
	sm.middlewareMap = make(map[string]*MiddlewareItem)
}

type BoardManifestList struct {
	XMLName       xml.Name         `xml:"board-manifest-list"`
	BoardManifest []*BoardManifest `xml:"board-manifest"`

	// Capture unknown tags and attributes
	Surprises []AnyTag   `xml:",any"`
	LostAttrs []xml.Attr `xml:",any,attr"`
}

type BoardManifest struct {
	XMLName       xml.Name `xml:"board-manifest"`
	DependencyURL string   `xml:"dependency-url,attr,omitempty"`
	CapabilityURL string   `xml:"capability-url,attr,omitempty"`
	URI           string   `xml:"uri"`
	Boards        *Boards

	// Capture unknown tags and attributes
	Surprises []AnyTag   `xml:",any"`
	LostAttrs []xml.Attr `xml:",any,attr"`
}

type AppManifestList struct {
	XMLName     xml.Name       `xml:"app-manifest-list"`
	AppManifest []*AppManifest `xml:"app-manifest"`

	// Capture unknown tags and attributes
	Surprises []AnyTag   `xml:",any"`
	LostAttrs []xml.Attr `xml:",any,attr"`
}

type AppManifest struct {
	XMLName xml.Name `xml:"app-manifest"`
	URI     string   `xml:"uri"`
	Apps    *Apps
	// Capture unknown tags and attributes
	Surprises []AnyTag   `xml:",any"`
	LostAttrs []xml.Attr `xml:",any,attr"`
}

type MiddlewareManifestList struct {
	XMLName            xml.Name              `xml:"middleware-manifest-list"`
	MiddlewareManifest []*MiddlewareManifest `xml:"middleware-manifest"`

	// Capture unknown tags and attributes
	Surprises []AnyTag   `xml:",any"`
	LostAttrs []xml.Attr `xml:",any,attr"`
}

type MiddlewareManifest struct {
	XMLName       xml.Name `xml:"middleware-manifest"`
	DependencyURL string   `xml:"dependency-url,attr,omitempty"`
	URI           string   `xml:"uri"`
	Middlewares   *Middleware

	// Capture unknown tags and attributes
	Surprises []AnyTag   `xml:",any"`
	LostAttrs []xml.Attr `xml:",any,attr"`
}

type Boards struct {
	XMLName xml.Name `xml:"boards"`
	Boards  []*Board `xml:"board"`

	// Capture unknown tags and attributes
	Surprises []AnyTag   `xml:",any"`
	LostAttrs []xml.Attr `xml:",any,attr"`
}

type Board struct {
	XMLName          xml.Name       `xml:"board"`
	ID               string         `xml:"id"`
	Category         string         `xml:"category"`
	BoardURI         string         `xml:"board_uri"`
	Chips            Chips          `xml:"chips"`
	Name             string         `xml:"name"`
	Summary          string         `xml:"summary"`
	ProvCapabilities string         `xml:"prov_capabilities"`
	Description      string         `xml:"description"`
	DocumentationURL string         `xml:"documentation_url"`
	Versions         *BoardVersions `xml:"versions"`
	DefaultLocation  string         `xml:"default_location,attr,omitempty"`

	//lint:ignore SA5008 Static checker false positive
	Origin *BoardManifest `json:"-" xml:"-"`
	//lint:ignore SA5008 Static checker false positive
	Dependencies *Depender                `xml:"-"`
	Capabilities *BSPCapabilitiesManifest `xml:"-"`

	// Capture unknown tags and attributes
	Surprises []AnyTag   `xml:",any"`
	LostAttrs []xml.Attr `xml:",any,attr"`
}

type Chips struct {
	XMLName xml.Name `xml:"chips"`
	MCU     []string `xml:"mcu"`
	Radio   []string `xml:"radio,omitempty"`

	// Capture unknown tags and attributes
	Surprises []AnyTag   `xml:",any"`
	LostAttrs []xml.Attr `xml:",any,attr"`
}
type BoardVersions struct {
	XMLName  xml.Name        `xml:"versions"`
	Versions []*BoardVersion `xml:"version"`

	// Capture unknown tags and attributes
	Surprises []AnyTag   `xml:",any"`
	LostAttrs []xml.Attr `xml:",any,attr"`
}

type BoardVersion struct {
	XMLName                    xml.Name `xml:"version"`
	FlowVersion                string   `xml:"flow_version,attr"`
	ProvCapabilitiesPerVersion string   `xml:"prov_capabilities_per_version,attr"`
	Num                        string   `xml:"num"`
	Commit                     string   `xml:"commit"`

	// Capture unknown tags and attributes
	Surprises []AnyTag   `xml:",any"`
	LostAttrs []xml.Attr `xml:",any,attr"`
}

// Middleware is the root element
type Middleware struct {
	XMLName     xml.Name          `xml:"middleware"`
	Middlewares []*MiddlewareItem `xml:"middleware"`

	// Capture unknown tags and attributes
	Surprises []AnyTag   `xml:",any"`
	LostAttrs []xml.Attr `xml:",any,attr"`
}

// MiddlewareItem represents a single middleware entry
type MiddlewareItem struct {
	XMLName           xml.Name    `xml:"middleware"`
	Type              string      `xml:"type,attr,omitempty"`
	Hidden            string      `xml:"hidden,attr,omitempty"`
	ReqCapabilitiesV2 string      `xml:"req_capabilities_v2,attr,omitempty"`
	Name              string      `xml:"n"`
	ID                string      `xml:"id"`
	URI               string      `xml:"uri"`
	Description       string      `xml:"desc"`
	Category          string      `xml:"category"`
	ReqCapabilities   string      `xml:"req_capabilities"`
	Versions          *MWVersions `xml:"versions"`
	//lint:ignore SA5008 Static checker false positive
	Origin *MiddlewareManifest `json:"-" xml:"-"`
	//lint:ignore SA5008 Static checker false positive
	Dependencies *Depender `xml:"-"`

	// Capture unknown tags and attributes
	Surprises []AnyTag   `xml:",any"`
	LostAttrs []xml.Attr `xml:",any,attr"`
}

// Versions contains a list of version entries
type MWVersions struct {
	XMLName xml.Name     `xml:"versions"`
	Version []*MWVersion `xml:"version"`

	// Capture unknown tags and attributes
	Surprises []AnyTag   `xml:",any"`
	LostAttrs []xml.Attr `xml:",any,attr"`
}

// Version represents a single version entry
type MWVersion struct {
	XMLName         xml.Name `xml:"version"`
	FlowVersion     string   `xml:"flow_version,attr,omitempty"`
	ToolsMinVersion string   `xml:"tools_min_version,attr,omitempty"`
	Num             string   `xml:"num"`
	Commit          string   `xml:"commit"`
	Desc            string   `xml:"desc"`

	// Capture unknown tags and attributes
	Surprises []AnyTag   `xml:",any"`
	LostAttrs []xml.Attr `xml:",any,attr"`
}

// CapabilitiesManifest is the root structure
type CapabilitiesManifest struct {
	Capabilities []*Capability `json:"capabilities"`
}

// Capability represents a single capability entry
type Capability struct {
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Name        string   `json:"name"`
	Token       string   `json:"token"`
	Types       []string `json:"types"`
}

// Code Example Manifest structures
// Handles both mtb-ce-manifest.xml (v1) and mtb-ce-manifest-fv2.xml (v2)

type Apps struct {
	XMLName xml.Name `xml:"apps"`
	Version string   `xml:"version,attr,omitempty"` // Only in v2 (fv2): "2.0"
	App     []*App   `xml:"app"`

	// Capture unknown tags and attributes
	Surprises []AnyTag   `xml:",any"`
	LostAttrs []xml.Attr `xml:",any,attr"`
}

type App struct {
	XMLName           xml.Name   `xml:"app"`
	Keywords          string     `xml:"keywords,attr,omitempty"`            // v2 only: comma-delimited
	ReqCapabilities   string     `xml:"req_capabilities,attr,omitempty"`    // v1: space-delimited string
	ReqCapabilitiesV2 string     `xml:"req_capabilities_v2,attr,omitempty"` // v2: bracketed syntax
	Name              string     `xml:"n"`
	ID                string     `xml:"id"`
	Category          string     `xml:"category,omitempty"` // v2 only
	URI               string     `xml:"uri"`
	Description       string     `xml:"description"`
	Versions          CEVersions `xml:"versions"`
	//lint:ignore SA5008 Static checker false positive
	Origin *AppManifest `json:"-" xml:"-"`

	// Capture unknown tags and attributes
	Surprises []AnyTag   `xml:",any"`
	LostAttrs []xml.Attr `xml:",any,attr"`
}

type CEVersions struct {
	XMLName xml.Name     `xml:"versions"`
	Version []*CEVersion `xml:"version"`

	// Capture unknown tags and attributes
	Surprises []AnyTag   `xml:",any"`
	LostAttrs []xml.Attr `xml:",any,attr"`
}

type CEVersion struct {
	XMLName                     xml.Name `xml:"version"`
	FlowVersion                 string   `xml:"flow_version,attr,omitempty"`
	ToolsMinVersion             string   `xml:"tools_min_version,attr,omitempty"`               // v2
	ToolsMaxVersion             string   `xml:"tools_max_version,attr,omitempty"`               // v1
	ReqCapabilitiesPerVersion   string   `xml:"req_capabilities_per_version,attr,omitempty"`    // v1: space-delimited
	ReqCapabilitiesPerVersionV2 string   `xml:"req_capabilities_per_version_v2,attr,omitempty"` // v2: bracketed syntax
	Num                         string   `xml:"num"`
	Commit                      string   `xml:"commit"`

	// Capture unknown tags and attributes
	Surprises []AnyTag   `xml:",any"`
	LostAttrs []xml.Attr `xml:",any,attr"`
}

func ReadSuperManifest(xmlData []byte) (*SuperManifest, error) {
	var superManifest SuperManifest
	err := UnmarshalXMLWithVerification(xmlData, &superManifest)
	if err != nil {
		return nil, err
	}
	return &superManifest, nil
}

func ReadBoardManifest(xmlData []byte) (*Boards, error) {
	var boards = Boards{}
	err := UnmarshalXMLWithVerification(xmlData, &boards)
	if err != nil {
		return nil, err
	}
	return &boards, nil
}

func ReadMiddlewareManifest(xmlData []byte) (*Middleware, error) {
	var middleware = Middleware{}
	err := UnmarshalXMLWithVerification(xmlData, &middleware)
	if err != nil {
		return nil, err
	}

	return &middleware, nil
}

// This is a lookup table mapping capability tokens (like "adc", "wifi", "cat1a") to
// their descriptions and categories. Looks like it's used to understand what
// features/hardware blocks are available on different boards and chips. Notice the
// types field indicates whether it applies to "chip", "board", or "generation".
func ReadCapabilitiesManifest(jsonData []byte) (*CapabilitiesManifest, error) {
	var manifest = CapabilitiesManifest{}
	err := json.Unmarshal(jsonData, &manifest)
	if err != nil {
		return nil, err
	}
	return &manifest, nil
}

// This manifest tells you "for BSP X version Y, you need these specific
// versions of libraries A, B, C, etc." Essential for dependency resolution
// and ensuring compatible versions are used together!
func ReadDependenciesManifest(xmlData []byte) (*Dependencies, error) {
	var deps = Dependencies{}
	err := UnmarshalXMLWithVerification(xmlData, &deps)
	if err != nil {
		return nil, err
	}
	return &deps, nil
}

func (manifest *SuperManifest) GetBoardsMap() *map[string]*Board {
	if (manifest.boardsMap != nil) && (len(manifest.boardsMap) > 0) {
		return &manifest.boardsMap
	}
	manifest.boardsMap = make(map[string]*Board)
	for _, bm := range manifest.BoardManifestList.BoardManifest {
		if bm.Boards != nil {
			for _, board := range bm.Boards.Boards {
				board.Origin = bm
				manifest.boardsMap[board.ID] = board
			}
		}
	}
	return &manifest.boardsMap
}

func (manifest *SuperManifest) GetBoardIDs() []string {
	boardIDs := []string{}
	for _, bm := range manifest.BoardManifestList.BoardManifest {
		if bm.Boards == nil {
			continue
		}
		for _, board := range bm.Boards.Boards {
			boardIDs = append(boardIDs, board.ID)
		}
	}
	return boardIDs
}

func (manifest *SuperManifest) GetBoard(boardID string) (*Board, bool) {
	boardsMap := manifest.GetBoardsMap()
	board, exists := (*boardsMap)[boardID]
	return board, exists
}

func (manifest *SuperManifest) GetAppsMap() *map[string]*App {
	if (manifest.appMap != nil) && (len(manifest.appMap) > 0) {
		return &manifest.appMap
	}
	manifest.appMap = make(map[string]*App)
	for _, am := range manifest.AppManifestList.AppManifest {
		if am.Apps != nil {
			for _, app := range am.Apps.App {
				app.Origin = am
				manifest.appMap[app.ID] = app
			}
		}
	}
	return &manifest.appMap
}

func (manifest *SuperManifest) GetAppIDs() []string {
	appIDs := []string{}
	for _, am := range manifest.AppManifestList.AppManifest {
		if am.Apps == nil {
			continue
		}
		for _, app := range am.Apps.App {
			appIDs = append(appIDs, app.ID)
		}
	}
	return appIDs
}

func (manifest *SuperManifest) GetApp(appID string) (*App, bool) {
	appsMap := manifest.GetAppsMap()
	app, exists := (*appsMap)[appID]
	return app, exists
}

func (manifest *SuperManifest) GetMiddlewareMap() *map[string]*MiddlewareItem {
	if (manifest.middlewareMap != nil) && (len(manifest.middlewareMap) > 0) {
		return &manifest.middlewareMap
	}
	manifest.middlewareMap = make(map[string]*MiddlewareItem)
	for _, mm := range manifest.MiddlewareManifestList.MiddlewareManifest {
		if mm.Middlewares != nil {
			for _, item := range mm.Middlewares.Middlewares {
				item.Origin = mm
				manifest.middlewareMap[item.ID] = item
			}
		}
	}
	return &manifest.middlewareMap
}

func (manifest *SuperManifest) GetMiddlewareIDs() []string {
	middlewareIDs := []string{}
	for _, mm := range manifest.MiddlewareManifestList.MiddlewareManifest {
		if mm.Middlewares == nil {
			continue
		}
		for _, item := range mm.Middlewares.Middlewares {
			middlewareIDs = append(middlewareIDs, item.ID)
		}
	}
	return middlewareIDs
}

func (manifest *SuperManifest) GetMiddleware(middlewareID string) (*MiddlewareItem, bool) {
	middlewareMap := manifest.GetMiddlewareMap()
	item, exists := (*middlewareMap)[middlewareID]
	return item, exists
}

// GetDependencies fetches and caches the BSP/Middleware dependencies manifest from the given URL
func (sm *SuperManifest) GetDependencies(urlStr string) *Dependencies {
	if (urlStr == "") || (urlStr == "N/A") {
		return nil
	}
	ret := sm.dependenciesMap[urlStr]
	return ret
}

func (sm *SuperManifest) GetBSPCapabilitiesManifest(urlStr string) *BSPCapabilitiesManifest {
	ret := sm.bspCapabilitiesMap[urlStr]
	return ret
}

// GetDependenciesByID retrieves the BSP dependencies for a specific BSP ID from the given URL
// Returns nil if the URL or ID is empty or "N/A"
func (sm *SuperManifest) GetDependenciesByID(urlStr string, Id string) *Depender {
	if (Id == "") || (Id == "N/A" || (urlStr == "") || (urlStr == "N/A")) {
		return nil
	}
	depManifest := sm.GetDependencies(urlStr)
	if depManifest != nil {
		return nil
	}
	return depManifest.GetBSP(Id)
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

func (sm *SuperManifest) AddSuperManifest(other *SuperManifest) {
	if (sm.Version != other.Version) && (other.Version != "") {
		// Should we error out instead?
		logger.Warningf("Merging super manifests with different versions: %s vs %s\n", sm.Version, other.Version)
	}
	sm.SourceUrls = append(sm.SourceUrls, other.SourceUrls...)
	// Merge Board Manifests
	sm.BoardManifestList.BoardManifest = append(sm.BoardManifestList.BoardManifest, other.BoardManifestList.BoardManifest...)
	// Merge App Manifests
	sm.AppManifestList.AppManifest = append(sm.AppManifestList.AppManifest, other.AppManifestList.AppManifest...)
	// Merge Middleware Manifests
	sm.MiddlewareManifestList.MiddlewareManifest = append(sm.MiddlewareManifestList.MiddlewareManifest, other.MiddlewareManifestList.MiddlewareManifest...)

	// If we have duplicate dependency or capability URLs, log a warning. It is possible
	// that we will have dangling references in board/middleware manifests, but that is up to the user
	// to resolve. It should not cause a crash. If this is a problem, we can enhance this to track
	// which manifest the URL came from and only warn if the same URL has different content.
	for k, v := range other.dependenciesMap {
		if _, exists := sm.dependenciesMap[k]; exists {
			logger.Warningf("Merging super manifests with duplicate dependency URL: %s\n", k)
		}
		sm.dependenciesMap[k] = v
	}
	for k, v := range other.bspCapabilitiesMap {
		if _, exists := sm.bspCapabilitiesMap[k]; exists {
			logger.Warningf("Merging super manifests with duplicate BSP capabilities URL: %s\n", k)
		}
		sm.bspCapabilitiesMap[k] = v
	}

	// Following maps will be rebuilt on demand. So, clear them instead of merging
	sm.clearMaps()
}

func (sm *SuperManifest) AddSuperManifestFromURL(urlStr string) error {
	otherManifest, err := NewSuperManifestFromURL(urlStr)
	if err != nil {
		return err
	}
	// Type assert to concrete type for internal merge operation
	if otherConcrete, ok := otherManifest.(*SuperManifest); ok {
		sm.AddSuperManifest(otherConcrete)
	}
	return nil
}

// IsV2 checks if this is a v2 format manifest
func (apps *Apps) IsV2() bool {
	return apps.Version == "2.0"
}

func ReadAppsManifest(data []byte) (*Apps, error) {
	var apps Apps
	if err := UnmarshalXMLWithVerification(data, &apps); err != nil {
		return nil, err
	}
	return &apps, nil
}

// GetKeywords returns the keywords as a slice, parsed from comma-delimited string
func (a *App) GetKeywords() []string {
	if a.Keywords == "" {
		return []string{}
	}

	keywords := strings.Split(a.Keywords, ",")
	result := make([]string, 0, len(keywords))
	for _, kw := range keywords {
		if trimmed := strings.TrimSpace(kw); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// GetToolsVersion returns the appropriate tools version string (min for v2, max for v1)
func (v *CEVersion) GetToolsVersion() (version string, isMin bool) {
	if v.ToolsMinVersion != "" {
		return v.ToolsMinVersion, true
	}
	return v.ToolsMaxVersion, false
}

// ////////////////////////////////////////////////////////////////////////
// XML Unmarshal verification
// ////////////////////////////////////////////////////////////////////////
var doVerifyXMLUnmarshal = false

// EnableXMLUnmarshalVerification enables or disables verification of XML unmarshaling
func EnableXMLUnmarshalVerification(enable bool) {
	if enable {
		logger.Infof("XML Unmarshal Verification Enabled\n")
	}
	doVerifyXMLUnmarshal = enable
}

func UnmarshalXMLWithVerification[T any](data []byte, obj *T) error {
	if err := xml.Unmarshal(data, obj); err != nil {
		return err
	}

	if doVerifyXMLUnmarshal {
		logger.Infof("End Unmarshal of Type %s, Begin Verification\n", reflect.TypeOf(*obj).Name())
		badPaths := FindDeepSurprisesInStruct(*obj)
		if len(badPaths) > 0 {
			for _, path := range badPaths {
				logger.Warningf("⚠️  XML Unmarshal Surprise: %s\n", path)
			}
		}
	}
	return nil
}
