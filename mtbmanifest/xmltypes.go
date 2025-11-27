package mtbmanifest

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"runtime"
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

	// GetBSPDependenciesManifest fetches and caches the BSP dependencies manifest from the given URL
	GetBSPDependenciesManifest(urlStr string) (*BSPDependenciesManifest, error)

	// GetBSPCapabilitiesManifest fetches and caches the BSP capabilities manifest from the given URL
	GetBSPCapabilitiesManifest(urlStr string) (*BSPCapabilitiesManifest, error)

	// GetBSPDependencies retrieves the BSP dependencies for a specific BSP ID from the given URL
	GetBSPDependencies(urlStr string, bspId string) (*BSPDepender, error)

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
	bspDependenciesMap map[string]*BSPDependenciesManifest
	bspCapabilitiesMap map[string]*BSPCapabilitiesManifest
}

// NewSuperManifest creates an empty SuperManifest ready to be populated.
func NewSuperManifest() SuperManifestIF {
	ret := &SuperManifest{
		BoardManifestList:      &BoardManifestList{},
		AppManifestList:        &AppManifestList{},
		MiddlewareManifestList: &MiddlewareManifestList{},
		bspDependenciesMap:     make(map[string]*BSPDependenciesManifest),
		bspCapabilitiesMap:     make(map[string]*BSPCapabilitiesManifest),
	}
	ret.clearMaps()
	return ret
}

// NewSuperManifestFromURL fetches and ingests a complete super manifest tree from the given URL.
// If urlStr is empty, it uses the default SuperManifestURL.
// This constructor fetches all board, app, and middleware manifests concurrently.
func NewSuperManifestFromURL(urlStr string) (SuperManifestIF, error) {
	urlFetcher := NewManifestFetcher(runtime.NumCPU())
	if urlStr == "" {
		urlStr = SuperManifestURL
	}

	logger.Infof("Fetching super manifest...%s\n", urlStr)
	superData, err := urlFetcher.Cache.Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch super manifest %s: %v", urlStr, err)
	}
	superManifest, err := UnmarshalManifest(superData, err, ReadSuperManifest)
	if err != nil {
		return nil, fmt.Errorf("failed to parse super manifest %s: %v", urlStr, err)
	}
	superManifest.SourceUrls = append(superManifest.SourceUrls, urlStr)
	superManifest.clearMaps()
	logger.Infof("Fetched super manifest with %d board manifests\n", len(superManifest.BoardManifestList.BoardManifest))

	urls := []FetchUrlWithCb{}
	var mu sync.Mutex
	for ix, mManifest := range superManifest.BoardManifestList.BoardManifest {
		item := FetchUrlWithCb{
			Url: mManifest.URI, Index: ix,
			Callback: func(urlStr string, data []byte, err error, index int) {
				logger.Infof("Board: %s: len=%d, err=%v, index=%d\n", urlStr, len(data), err, index)
				boards, err := UnmarshalManifest(data, err, ReadBoardManifest)
				if err != nil {
					logger.Errorf("Error fetching %s: %v\n", urlStr, err)
				} else {
					mu.Lock()
					superManifest.BoardManifestList.BoardManifest[index].Boards = boards
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
				logger.Infof("App: %s: len=%d, err=%v, index=%d\n", urlStr, len(data), err, index)
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
		item := FetchUrlWithCb{
			Url: mManifest.URI, Index: ix,
			Callback: func(urlStr string, data []byte, err error, index int) {
				logger.Infof("Middleware: %s: len=%d, err=%v, index=%d\n", urlStr, len(data), err, index)
				middleware, err := UnmarshalManifest(data, err, ReadMiddlewareManifest)
				if err != nil {
					logger.Errorf("Error fetching file %s: %v\n", urlStr, err)
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

// Maps are cleared when manifests are merged or modified so that they can be rebuilt on demand
func (sm *SuperManifest) clearMaps() {
	sm.boardsMap = make(map[string]*Board)
	sm.appMap = make(map[string]*App)
	sm.middlewareMap = make(map[string]*MiddlewareItem)
}

type BoardManifestList struct {
	XMLName       xml.Name         `xml:"board-manifest-list"`
	BoardManifest []*BoardManifest `xml:"board-manifest"`
}

type BoardManifest struct {
	XMLName       xml.Name `xml:"board-manifest"`
	DependencyURL string   `xml:"dependency-url,attr,omitempty"`
	CapabilityURL string   `xml:"capability-url,attr,omitempty"`
	URI           string   `xml:"uri"`
	Boards        *Boards
}

type AppManifestList struct {
	XMLName     xml.Name       `xml:"app-manifest-list"`
	AppManifest []*AppManifest `xml:"app-manifest"`
}

type AppManifest struct {
	XMLName xml.Name `xml:"app-manifest"`
	URI     string   `xml:"uri"`
	Apps    *Apps
}

type MiddlewareManifestList struct {
	XMLName            xml.Name              `xml:"middleware-manifest-list"`
	MiddlewareManifest []*MiddlewareManifest `xml:"middleware-manifest"`
}

type MiddlewareManifest struct {
	XMLName       xml.Name `xml:"middleware-manifest"`
	DependencyURL string   `xml:"dependency-url,attr,omitempty"`
	URI           string   `xml:"uri"`
	Middlewares   *Middleware
}

type Boards struct {
	XMLName xml.Name `xml:"boards"`
	Boards  []Board  `xml:"board"`
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
	Origin          *BoardManifest `json:"-" xml:"-"`
	BSPDependencies *BSPDepender
	BSPCapabilities *BSPCapabilitiesManifest
}

type Chips struct {
	XMLName xml.Name `xml:"chips"`
	MCU     []string `xml:"mcu"`
	Radio   []string `xml:"radio,omitempty"`
}

type BoardVersions struct {
	XMLName  xml.Name        `xml:"versions"`
	Versions []*BoardVersion `xml:"version"`
}

type BoardVersion struct {
	XMLName                    xml.Name `xml:"version"`
	FlowVersion                string   `xml:"flow_version,attr"`
	ProvCapabilitiesPerVersion string   `xml:"prov_capabilities_per_version,attr"`
	Num                        string   `xml:"num"`
	Commit                     string   `xml:"commit"`
}

// Middleware is the root element
type Middleware struct {
	XMLName     xml.Name          `xml:"middleware"`
	Middlewares []*MiddlewareItem `xml:"middleware"`
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
}

// Versions contains a list of version entries
type MWVersions struct {
	XMLName xml.Name     `xml:"versions"`
	Version []*MWVersion `xml:"version"`
}

// Version represents a single version entry
type MWVersion struct {
	XMLName         xml.Name `xml:"version"`
	FlowVersion     string   `xml:"flow_version,attr,omitempty"`
	ToolsMinVersion string   `xml:"tools_min_version,attr,omitempty"`
	Num             string   `xml:"num"`
	Commit          string   `xml:"commit"`
	Desc            string   `xml:"desc"`
}

// Dependencies is the root element
type Dependencies struct {
	XMLName  xml.Name    `xml:"dependencies"`
	Version  string      `xml:"version,attr"`
	Depender []*Depender `xml:"depender"`
}

// Depender represents a BSP that has dependencies
type Depender struct {
	XMLName  xml.Name            `xml:"depender"`
	ID       string              `xml:"id"`
	Versions *DependencyVersions `xml:"versions"`
}

// DependencyVersions contains version-specific dependency information
type DependencyVersions struct {
	XMLName xml.Name             `xml:"versions"`
	Version []*DependencyVersion `xml:"version"`
}

// DependencyVersion represents dependencies for a specific version
type DependencyVersion struct {
	XMLName   xml.Name   `xml:"version"`
	Commit    string     `xml:"commit"`
	Dependees *Dependees `xml:"dependees"`
}

// Dependees is a container for all the dependencies
type Dependees struct {
	XMLName  xml.Name    `xml:"dependees"`
	Dependee []*Dependee `xml:"dependee"`
}

// Dependee represents a single dependency (library/middleware)
type Dependee struct {
	XMLName xml.Name `xml:"dependee"`
	ID      string   `xml:"id"`
	Commit  string   `xml:"commit"`
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

func ReadSuperManifest(xmlData []byte) (*SuperManifest, error) {
	var superManifest SuperManifest
	err := xml.Unmarshal(xmlData, &superManifest)
	if err != nil {
		return nil, err
	}
	return &superManifest, nil
}

func ReadBoardManifest(xmlData []byte) (*Boards, error) {
	var boards = Boards{}
	err := xml.Unmarshal(xmlData, &boards)
	if err != nil {
		return nil, err
	}
	return &boards, nil
}

func ReadMiddlewareManifest(xmlData []byte) (*Middleware, error) {
	var middleware = Middleware{}
	err := xml.Unmarshal(xmlData, &middleware)
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
	err := xml.Unmarshal(xmlData, &deps)
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
				manifest.boardsMap[board.ID] = &board
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
				manifest.appMap[app.ID] = &app
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

func (sm *SuperManifest) GetBSPDependenciesManifest(urlStr string) (*BSPDependenciesManifest, error) {
	if (urlStr == "") || (urlStr == "N/A") {
		return nil, nil
	}
	ret := sm.bspDependenciesMap[urlStr]
	if ret != nil {
		return ret, nil
	}
	if sm.bspDependenciesMap == nil {
		sm.bspDependenciesMap = make(map[string]*BSPDependenciesManifest)
	}
	mC := NewManifestDefaultCache()
	data, err := mC.Get(urlStr)
	if err != nil {
		return nil, err
	}
	depManifest, err := ReadBSPDependenciesManifest(data)
	if err != nil {
		return nil, err
	}
	sm.bspDependenciesMap[urlStr] = depManifest
	return depManifest, nil
}

func (sm *SuperManifest) GetBSPCapabilitiesManifest(urlStr string) (*BSPCapabilitiesManifest, error) {
	ret := sm.bspCapabilitiesMap[urlStr]
	if ret != nil {
		return ret, nil
	}
	if sm.bspCapabilitiesMap == nil {
		sm.bspCapabilitiesMap = make(map[string]*BSPCapabilitiesManifest)
	}
	mC := NewManifestDefaultCache()
	data, err := mC.Get(urlStr)
	if err != nil {
		return nil, err
	}
	depManifest, err := ReadBSPCapabilitiesManifest(data)
	if err != nil {
		return nil, err
	}
	sm.bspCapabilitiesMap[urlStr] = depManifest
	return depManifest, nil
}

func (sm *SuperManifest) GetBSPDependencies(urlStr string, bspId string) (*BSPDepender, error) {
	if (bspId == "") || (bspId == "N/A" || (urlStr == "") || (urlStr == "N/A")) {
		return nil, nil
	}
	depManifest, err := sm.GetBSPDependenciesManifest(urlStr)
	if err != nil {
		return nil, err
	}
	return depManifest.GetBSP(bspId), nil
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
	for k, v := range other.bspDependenciesMap {
		sm.bspDependenciesMap[k] = v
	}
	for k, v := range other.bspCapabilitiesMap {
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
