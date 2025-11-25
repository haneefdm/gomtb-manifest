package mtbmanifest

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
)

type Boards struct {
	XMLName xml.Name `xml:"boards"`
	Boards  []Board  `xml:"board"`
}

type Board struct {
	XMLName          xml.Name      `xml:"board"`
	ID               string        `xml:"id"`
	Category         string        `xml:"category"`
	BoardURI         string        `xml:"board_uri"`
	Chips            Chips         `xml:"chips"`
	Name             string        `xml:"name"`
	Summary          string        `xml:"summary"`
	ProvCapabilities string        `xml:"prov_capabilities"`
	Description      string        `xml:"description"`
	DocumentationURL string        `xml:"documentation_url"`
	Versions         BoardVersions `xml:"versions"`
	DefaultLocation  string        `xml:"default_location,attr,omitempty"`
}

type Chips struct {
	XMLName xml.Name `xml:"chips"`
	MCU     []string `xml:"mcu"`
	Radio   []string `xml:"radio,omitempty"`
}

type BoardVersions struct {
	XMLName  xml.Name       `xml:"versions"`
	Versions []BoardVersion `xml:"version"`
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
	XMLName    xml.Name         `xml:"middleware"`
	Middleware []MiddlewareItem `xml:"middleware"`
}

// MiddlewareItem represents a single middleware entry
type MiddlewareItem struct {
	XMLName           xml.Name   `xml:"middleware"`
	Type              string     `xml:"type,attr,omitempty"`
	Hidden            string     `xml:"hidden,attr,omitempty"`
	ReqCapabilitiesV2 string     `xml:"req_capabilities_v2,attr,omitempty"`
	Name              string     `xml:"n"`
	ID                string     `xml:"id"`
	URI               string     `xml:"uri"`
	Description       string     `xml:"desc"`
	Category          string     `xml:"category"`
	ReqCapabilities   string     `xml:"req_capabilities"`
	Versions          MWVersions `xml:"versions"`
}

// Versions contains a list of version entries
type MWVersions struct {
	XMLName xml.Name    `xml:"versions"`
	Version []MWVersion `xml:"version"`
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
	XMLName  xml.Name   `xml:"dependencies"`
	Version  string     `xml:"version,attr"`
	Depender []Depender `xml:"depender"`
}

// Depender represents a BSP that has dependencies
type Depender struct {
	XMLName  xml.Name           `xml:"depender"`
	ID       string             `xml:"id"`
	Versions DependencyVersions `xml:"versions"`
}

// DependencyVersions contains version-specific dependency information
type DependencyVersions struct {
	XMLName xml.Name            `xml:"versions"`
	Version []DependencyVersion `xml:"version"`
}

// DependencyVersion represents dependencies for a specific version
type DependencyVersion struct {
	XMLName   xml.Name  `xml:"version"`
	Commit    string    `xml:"commit"`
	Dependees Dependees `xml:"dependees"`
}

// Dependees is a container for all the dependencies
type Dependees struct {
	XMLName  xml.Name   `xml:"dependees"`
	Dependee []Dependee `xml:"dependee"`
}

// Dependee represents a single dependency (library/middleware)
type Dependee struct {
	XMLName xml.Name `xml:"dependee"`
	ID      string   `xml:"id"`
	Commit  string   `xml:"commit"`
}

// CapabilitiesManifest is the root structure
type CapabilitiesManifest struct {
	Capabilities []Capability `json:"capabilities"`
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
// Handles both mtb-ce-manifest.xml and mtb-ce-manifest-fv2.xml
type CEApps struct {
	XMLName xml.Name `xml:"apps"`
	Version string   `xml:"version,attr,omitempty"` // Only in v2 (fv2)
	App     []CEApp  `xml:"app"`
}

type CEApp struct {
	XMLName           xml.Name   `xml:"app"`
	Keywords          string     `xml:"keywords,attr,omitempty"`            // v2 only
	ReqCapabilities   string     `xml:"req_capabilities,attr,omitempty"`    // v1: space-delimited string
	ReqCapabilitiesV2 string     `xml:"req_capabilities_v2,attr,omitempty"` // v2: bracketed syntax like "[psoc6,wifi] [std_crypto]"
	Name              string     `xml:"n"`
	ID                string     `xml:"id"`
	Category          string     `xml:"category,omitempty"` // v2 only
	URI               string     `xml:"uri"`
	Description       string     `xml:"description"`
	Versions          CEVersions `xml:"versions"`
}

type CEVersions struct {
	XMLName xml.Name    `xml:"versions"`
	Version []CEVersion `xml:"version"`
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
}

func ReadSuperManifest(xmlData []byte) (*Boards, error) {
	var boards Boards
	err := xml.Unmarshal(xmlData, &boards)
	if err != nil {
		return Boards{}, err
	}

	// Access the data
	for _, board := range boards.Boards {
		fmt.Println(board.Name, board.ID)
	}
	return &boards, nil
}

func ReadMiddlewareManifest(xmlData []byte) (*Middleware, error) {
	var middleware Middleware
	err := xml.Unmarshal(xmlData, &middleware)
	if err != nil {
		return nil, err
	}

	// Access middleware items
	for _, item := range mw.Middleware {
		fmt.Printf("Middleware: %s (ID: %s)\n", item.Name, item.ID)
		fmt.Printf("  Category: %s\n", item.Category)
		fmt.Printf("  Versions: %d\n", len(item.Versions.Version))
	}
	return &middleware, nil
}

// This is a lookup table mapping capability tokens (like "adc", "wifi", "cat1a") to
// their descriptions and categories. Looks like it's used to understand what
// features/hardware blocks are available on different boards and chips. Notice the
// types field indicates whether it applies to "chip", "board", or "generation".
func ReadCapabilitiesManifest(jsonData []byte) (*CapabilitiesManifest, error) {
	var manifest CapabilitiesManifest
	err := json.Unmarshal(jsonData, &manifest)
	if err != nil {
		return nil, err
	}

	// Access capabilities
	for _, cap := range manifest.Capabilities {
		fmt.Printf("Capability: %s (%s)\n", cap.Name, cap.Token)
		fmt.Printf("  Category: %s\n", cap.Category)
		fmt.Printf("  Types: %v\n", cap.Types)
	}
	return &manifest, nil
}

// This manifest tells you "for BSP X version Y, you need these specific
// versions of libraries A, B, C, etc." Essential for dependency resolution
// and ensuring compatible versions are used together!
func ReadDependenciesManifest(xmlData []byte) (*Dependencies, error) {
	var dependencies Dependencies
	err := xml.Unmarshal(xmlData, &dependencies)
	if err != nil {
		return nil, err
	}

	// Access dependencies
	for _, depender := range deps.Depender {
		fmt.Printf("BSP: %s\n", depender.ID)
		for _, ver := range depender.Versions.Version {
			fmt.Printf("  Version: %s\n", ver.Commit)
			fmt.Printf("    Dependencies:\n")
			for _, dep := range ver.Dependees.Dependee {
				fmt.Printf("      - %s @ %s\n", dep.ID, dep.Commit)
			}
		}
	}
	return &dependencies, nil
}

/*
Key differences to note:

Root element: v2 has <apps version="2.0">, v1 just has <apps>
Capability syntax:

v1: req_capabilities="psoc6 led" (space-delimited string)
v2: req_capabilities_v2="[psoc6,t2gbe] hal led" (bracketed groups with commas, OR logic within brackets)

New v2 fields:

keywords attribute on <app>
<category> element
tools_min_version instead of tools_max_version

Per-version capabilities also have v1/v2 variants with similar syntax differences

The bracketed syntax in v2 appears to support complex boolean logic where:

Items in brackets [a,b,c] are OR'd together
Multiple bracket groups are AND'd together
Plain items are required

Example: req_capabilities_v2="[psoc6,t2gbe] hal led [flash_2048k,flash_1024k]" means:

(psoc6 OR t2gbe) AND hal AND led AND (flash_2048k OR flash_1024k)
*/
func ReadCEManifest(xmlData []byte) (CEApps, error) {
	var ceApps CEApps
	err := xml.Unmarshal(xmlData, &ceApps)
	if err != nil {
		return CEApps{}, err
	}

	// Access CE apps
	for _, app := range ceApps.App {
		fmt.Printf("App: %s (ID: %s)\n", app.Name, app.ID)
		fmt.Printf("  Versions: %d\n", len(app.Versions.Version))
	}
	return ceApps, nil
}
