package mtbmanifest

import (
	"encoding/xml"
)

// This is a generic way to represent board and middleware dependencies in MTB manifest XML files.
// dependencies_manifest.go - shared by both BSP and MW
// Example URL: https://raw.githubusercontent.com/Infineon/mtb-bsp-manifest/v2.X/mtb-bsp-dependencies-manifest.xml
type Dependencies struct {
	XMLName   xml.Name    `xml:"dependencies"`
	Version   string      `xml:"version,attr"`
	Dependers []*Depender `xml:"depender"`

	// Capture unknown tags and attributes
	Surprises []AnyTag   `xml:",any"`
	LostAttrs []xml.Attr `xml:",any,attr"`

	// Following maps are created post-unmarshal for easy lookup
	// DependersMap maps depender IDs to Depender structs for quick lookup
	DependersMap map[string]*Depender `xml:"-"`
	// LibraryMap maps library IDs to the list of BSP IDs that depend on them
	LibraryMap map[string][]string `xml:"-"`
}

type Depender struct {
	XMLName  xml.Name           `xml:"depender"`
	ID       string             `xml:"id"`
	Versions []*DependerVersion `xml:"versions>version"`

	// Capture unknown tags and attributes
	Surprises []AnyTag   `xml:",any"`
	LostAttrs []xml.Attr `xml:",any,attr"`

	// Following map is created post-unmarshal for easy lookup
	VersionsMap map[string]*DependerVersion `xml:"-"`
}

type DependerVersion struct {
	Commit    string      `xml:"commit"`
	Dependees []*Dependee `xml:"dependees>dependee"`

	// Capture unknown tags and attributes
	Surprises []AnyTag   `xml:",any"`
	LostAttrs []xml.Attr `xml:",any,attr"`

	// Following map is created post-unmarshal for easy lookup
	DependeesMap map[string]*Dependee `xml:"-"`
}

type Dependee struct {
	ID     string `xml:"id"`
	Commit string `xml:"commit"`

	// Capture unknown attributes
	LostAttrs []xml.Attr `xml:",any,attr"`
}

// User wants to add bluetooth-freertos to their project
func ResolveDependencies(mwDeps *Dependencies, libraryID, version string) []string {
	var allDeps []string
	visited := make(map[string]bool)

	var resolve func(id, ver string)
	resolve = func(id, ver string) {
		if visited[id] {
			return
		}
		visited[id] = true
		allDeps = append(allDeps, id)

		// Get dependencies of this library
		deps, found := mwDeps.GetDependencies(id, ver)
		if !found {
			return
		}

		// Recursively resolve
		for _, dep := range deps {
			resolve(dep.ID, dep.Commit)
		}
	}

	resolve(libraryID, version)
	return allDeps
}

// Usage:
//    allDeps := ResolveDependencies(&mwDeps, "bluetooth-freertos", "latest-v3.X")
// Returns: ["bluetooth-freertos", "btstack", "freertos", "abstraction-rtos", "clib-support"]
