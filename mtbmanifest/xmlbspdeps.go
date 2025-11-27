package mtbmanifest

import "encoding/xml"

// BSPDependenciesManifest represents the root dependencies manifest structure
// Example URL: https://raw.githubusercontent.com/Infineon/mtb-bsp-manifest/v2.X/mtb-bsp-dependencies-manifest.xml
type BSPDependenciesManifest struct {
	XMLName xml.Name `xml:"dependencies"`
	Version string   `xml:"version,attr"`

	// Dependers are the BSPs (boards) that have dependencies
	Dependers    []*BSPDepender          `xml:"depender"`
	DependersMap map[string]*BSPDepender `xml:"-"` // Map of BSP ID to BSPDepender for quick lookup

	// LibraryMap maps library IDs to the list of BSP IDs that depend on them
	LibraryMap map[string][]string `xml:"-"`
}

// BSPDepender represents a BSP (board) and its version-specific dependencies
type BSPDepender struct {
	// ID is the BSP identifier (e.g., "CY8CKIT-040T", "PMG1-CY7110")
	ID string `xml:"id"`

	// Versions contains the dependency information for each version of this BSP
	Versions    []*BSPDependerVersion          `xml:"versions>version"`
	VersionsMap map[string]*BSPDependerVersion `xml:"-"`
}

// BSPDependerVersion represents dependencies for a specific version of a BSP
type BSPDependerVersion struct {
	// Commit is the version/tag for this BSP (e.g., "latest-v3.X", "release-v3.2.0")
	Commit string `xml:"commit"`

	// Dependees are the libraries/components this BSP version depends on
	Dependees    []*BSPDependee          `xml:"dependees>dependee"`
	DependeesMap map[string]*BSPDependee `xml:"-"`
}

// BSPDependee represents a library or component that a BSP depends on
type BSPDependee struct {
	// ID is the library identifier (e.g., "core-lib", "mtb-pdl-cat2", "recipe-make-cat2")
	ID string `xml:"id"`

	// Commit is the required version/tag of this dependency (e.g., "latest-v1.X", "latest-v2.X")
	Commit string `xml:"commit"`
}

func (m *BSPDependenciesManifest) CreateMaps() map[string]*BSPDepender {
	if m.DependersMap == nil {
		m.DependersMap = make(map[string]*BSPDepender)
		m.LibraryMap = make(map[string][]string)
		for _, depender := range m.Dependers {
			// depender.ID is the BSP ID
			m.DependersMap[depender.ID] = depender
			depender.VersionsMap = make(map[string]*BSPDependerVersion)
			for _, v := range depender.Versions {
				depender.VersionsMap[v.Commit] = v
				v.DependeesMap = make(map[string]*BSPDependee)
				for _, dependee := range v.Dependees {
					// dependee.ID is the library ID
					v.DependeesMap[dependee.ID] = dependee
					m.LibraryMap[dependee.ID] = append(m.LibraryMap[dependee.ID], depender.ID)
				}
			}
		}
	}
	return m.DependersMap
}

// Helper function to get dependencies for a specific BSP and version
func (m *BSPDependenciesManifest) GetDependencies(bspID, version string) ([]*BSPDependee, bool) {
	if depender, exists := m.CreateMaps()[bspID]; exists {
		if versionEntry, exists := depender.VersionsMap[version]; exists {
			return versionEntry.Dependees, true
		}
	}
	return nil, false
}

func (m *BSPDependenciesManifest) GetBSP(bspID string) *BSPDepender {
	return m.CreateMaps()[bspID]
}

// Helper function to get all versions of a BSP
func (m *BSPDependenciesManifest) GetBSPVersions(bspID string) ([]*BSPDependerVersion, map[string]*BSPDependerVersion, bool) {
	if depender, exists := m.CreateMaps()[bspID]; exists {
		return depender.Versions, depender.VersionsMap, true
	}
	return nil, nil, false
}

// Helper function to get all BSPs
func (m *BSPDependenciesManifest) GetAllBSPs() []string {
	bsps := make([]string, len(m.Dependers))
	for i, depender := range m.Dependers {
		bsps[i] = depender.ID
	}
	return bsps
}

// Helper function to find all BSPs that depend on a specific library
func (m *BSPDependenciesManifest) FindBSPsUsingLibrary(libraryID string) []string {
	_ = m.CreateMaps()
	return m.LibraryMap[libraryID]
}

func ReadBSPDependenciesManifest(data []byte) (*BSPDependenciesManifest, error) {
	var manifest BSPDependenciesManifest
	err := xml.Unmarshal(data, &manifest)
	if err != nil {
		return nil, err
	}
	return &manifest, nil
}
