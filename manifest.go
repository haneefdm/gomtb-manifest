package gomtb

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
)

// Manifest represents the root structure of an MTB manifest file
type Manifest struct {
	XMLName      xml.Name     `xml:"manifest"`
	Name         string       `xml:"name,attr"`
	Version      string       `xml:"version,attr"`
	Description  string       `xml:"description"`
	Dependencies []Dependency `xml:"dependencies>dependency"`
	Boards       []Board      `xml:"boards>board"`
	Apps         []App        `xml:"apps>app"`
}

// Dependency represents a dependency entry in the manifest
type Dependency struct {
	Name     string `xml:"name,attr"`
	Version  string `xml:"version,attr"`
	URI      string `xml:"uri,attr"`
	Required bool   `xml:"required,attr"`
}

// Board represents a board support package in the manifest
type Board struct {
	Name        string   `xml:"name,attr"`
	Version     string   `xml:"version,attr"`
	Description string   `xml:"description"`
	Chips       []string `xml:"chips>chip"`
}

// App represents an application entry in the manifest
type App struct {
	Name        string `xml:"name,attr"`
	Version     string `xml:"version,attr"`
	Description string `xml:"description"`
	Path        string `xml:"path,attr"`
}

// ParseManifest parses an MTB manifest from a reader
func ParseManifest(r io.Reader) (*Manifest, error) {
	var manifest Manifest
	decoder := xml.NewDecoder(r)
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to decode manifest: %w", err)
	}
	return &manifest, nil
}

// ParseManifestFile parses an MTB manifest from a file path
func ParseManifestFile(path string) (*Manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest file: %w", err)
	}
	defer file.Close()

	return ParseManifest(file)
}

// ToXML converts a Manifest back to XML format
func (m *Manifest) ToXML() ([]byte, error) {
	output, err := xml.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}
	// Add XML header
	xmlHeader := []byte(xml.Header)
	return append(xmlHeader, output...), nil
}

// WriteToFile writes the manifest to a file
func (m *Manifest) WriteToFile(path string) error {
	xmlData, err := m.ToXML()
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, xmlData, 0644); err != nil {
		return fmt.Errorf("failed to write manifest file: %w", err)
	}
	return nil
}

// GetDependency finds a dependency by name
func (m *Manifest) GetDependency(name string) *Dependency {
	for i := range m.Dependencies {
		if m.Dependencies[i].Name == name {
			return &m.Dependencies[i]
		}
	}
	return nil
}

// GetBoard finds a board by name
func (m *Manifest) GetBoard(name string) *Board {
	for i := range m.Boards {
		if m.Boards[i].Name == name {
			return &m.Boards[i]
		}
	}
	return nil
}

// GetApp finds an app by name
func (m *Manifest) GetApp(name string) *App {
	for i := range m.Apps {
		if m.Apps[i].Name == name {
			return &m.Apps[i]
		}
	}
	return nil
}

// AddDependency adds a new dependency to the manifest
func (m *Manifest) AddDependency(dep Dependency) {
	m.Dependencies = append(m.Dependencies, dep)
}

// AddBoard adds a new board to the manifest
func (m *Manifest) AddBoard(board Board) {
	m.Boards = append(m.Boards, board)
}

// AddApp adds a new app to the manifest
func (m *Manifest) AddApp(app App) {
	m.Apps = append(m.Apps, app)
}
