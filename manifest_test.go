package gomtb

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleManifest = `<?xml version="1.0" encoding="UTF-8"?>
<manifest name="test-manifest" version="1.0.0">
  <description>Test MTB Manifest</description>
  <dependencies>
    <dependency name="core-lib" version="1.2.3" uri="https://github.com/example/core-lib" required="true"/>
    <dependency name="helper-lib" version="2.0.0" uri="https://github.com/example/helper-lib" required="false"/>
  </dependencies>
  <boards>
    <board name="CY8CPROTO-062-4343W" version="2.5.0">
      <description>PSoC 6 Wi-Fi BT Prototyping Kit</description>
      <chips>
        <chip>PSoC6</chip>
        <chip>CYW4343W</chip>
      </chips>
    </board>
  </boards>
  <apps>
    <app name="hello-world" version="1.0.0" path="apps/hello-world">
      <description>Hello World Application</description>
    </app>
  </apps>
</manifest>`

func TestParseManifest(t *testing.T) {
	reader := strings.NewReader(sampleManifest)
	manifest, err := ParseManifest(reader)
	if err != nil {
		t.Fatalf("Failed to parse manifest: %v", err)
	}

	// Test basic attributes
	if manifest.Name != "test-manifest" {
		t.Errorf("Expected name 'test-manifest', got '%s'", manifest.Name)
	}
	if manifest.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", manifest.Version)
	}
	if manifest.Description != "Test MTB Manifest" {
		t.Errorf("Expected description 'Test MTB Manifest', got '%s'", manifest.Description)
	}

	// Test dependencies
	if len(manifest.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(manifest.Dependencies))
	}
	if manifest.Dependencies[0].Name != "core-lib" {
		t.Errorf("Expected first dependency name 'core-lib', got '%s'", manifest.Dependencies[0].Name)
	}
	if manifest.Dependencies[0].Version != "1.2.3" {
		t.Errorf("Expected first dependency version '1.2.3', got '%s'", manifest.Dependencies[0].Version)
	}
	if !manifest.Dependencies[0].Required {
		t.Error("Expected first dependency to be required")
	}

	// Test boards
	if len(manifest.Boards) != 1 {
		t.Errorf("Expected 1 board, got %d", len(manifest.Boards))
	}
	if manifest.Boards[0].Name != "CY8CPROTO-062-4343W" {
		t.Errorf("Expected board name 'CY8CPROTO-062-4343W', got '%s'", manifest.Boards[0].Name)
	}
	if len(manifest.Boards[0].Chips) != 2 {
		t.Errorf("Expected 2 chips, got %d", len(manifest.Boards[0].Chips))
	}

	// Test apps
	if len(manifest.Apps) != 1 {
		t.Errorf("Expected 1 app, got %d", len(manifest.Apps))
	}
	if manifest.Apps[0].Name != "hello-world" {
		t.Errorf("Expected app name 'hello-world', got '%s'", manifest.Apps[0].Name)
	}
}

func TestParseManifestFile(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test-manifest.xml")

	if err := os.WriteFile(tmpFile, []byte(sampleManifest), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	manifest, err := ParseManifestFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to parse manifest file: %v", err)
	}

	if manifest.Name != "test-manifest" {
		t.Errorf("Expected name 'test-manifest', got '%s'", manifest.Name)
	}
}

func TestParseManifestFile_FileNotFound(t *testing.T) {
	_, err := ParseManifestFile("/nonexistent/path/manifest.xml")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestToXML(t *testing.T) {
	reader := strings.NewReader(sampleManifest)
	manifest, err := ParseManifest(reader)
	if err != nil {
		t.Fatalf("Failed to parse manifest: %v", err)
	}

	xmlData, err := manifest.ToXML()
	if err != nil {
		t.Fatalf("Failed to convert to XML: %v", err)
	}

	if !bytes.Contains(xmlData, []byte("test-manifest")) {
		t.Error("XML output should contain manifest name")
	}
	if !bytes.Contains(xmlData, []byte("<?xml version")) {
		t.Error("XML output should contain XML header")
	}
}

func TestWriteToFile(t *testing.T) {
	reader := strings.NewReader(sampleManifest)
	manifest, err := ParseManifest(reader)
	if err != nil {
		t.Fatalf("Failed to parse manifest: %v", err)
	}

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "output-manifest.xml")

	if err := manifest.WriteToFile(tmpFile); err != nil {
		t.Fatalf("Failed to write manifest file: %v", err)
	}

	// Verify file was created and can be read back
	manifest2, err := ParseManifestFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read back manifest: %v", err)
	}

	if manifest2.Name != manifest.Name {
		t.Errorf("Expected name '%s', got '%s'", manifest.Name, manifest2.Name)
	}
}

func TestGetDependency(t *testing.T) {
	reader := strings.NewReader(sampleManifest)
	manifest, err := ParseManifest(reader)
	if err != nil {
		t.Fatalf("Failed to parse manifest: %v", err)
	}

	dep := manifest.GetDependency("core-lib")
	if dep == nil {
		t.Fatal("Expected to find 'core-lib' dependency")
	}
	if dep.Version != "1.2.3" {
		t.Errorf("Expected version '1.2.3', got '%s'", dep.Version)
	}

	nonExistent := manifest.GetDependency("nonexistent")
	if nonExistent != nil {
		t.Error("Expected nil for nonexistent dependency")
	}
}

func TestGetBoard(t *testing.T) {
	reader := strings.NewReader(sampleManifest)
	manifest, err := ParseManifest(reader)
	if err != nil {
		t.Fatalf("Failed to parse manifest: %v", err)
	}

	board := manifest.GetBoard("CY8CPROTO-062-4343W")
	if board == nil {
		t.Fatal("Expected to find board")
	}
	if board.Version != "2.5.0" {
		t.Errorf("Expected version '2.5.0', got '%s'", board.Version)
	}

	nonExistent := manifest.GetBoard("nonexistent")
	if nonExistent != nil {
		t.Error("Expected nil for nonexistent board")
	}
}

func TestGetApp(t *testing.T) {
	reader := strings.NewReader(sampleManifest)
	manifest, err := ParseManifest(reader)
	if err != nil {
		t.Fatalf("Failed to parse manifest: %v", err)
	}

	app := manifest.GetApp("hello-world")
	if app == nil {
		t.Fatal("Expected to find 'hello-world' app")
	}
	if app.Path != "apps/hello-world" {
		t.Errorf("Expected path 'apps/hello-world', got '%s'", app.Path)
	}

	nonExistent := manifest.GetApp("nonexistent")
	if nonExistent != nil {
		t.Error("Expected nil for nonexistent app")
	}
}

func TestAddDependency(t *testing.T) {
	manifest := &Manifest{
		Name:    "test",
		Version: "1.0.0",
	}

	newDep := Dependency{
		Name:     "new-lib",
		Version:  "3.0.0",
		URI:      "https://example.com/new-lib",
		Required: true,
	}

	manifest.AddDependency(newDep)

	if len(manifest.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(manifest.Dependencies))
	}
	if manifest.Dependencies[0].Name != "new-lib" {
		t.Errorf("Expected dependency name 'new-lib', got '%s'", manifest.Dependencies[0].Name)
	}
}

func TestAddBoard(t *testing.T) {
	manifest := &Manifest{
		Name:    "test",
		Version: "1.0.0",
	}

	newBoard := Board{
		Name:        "TEST-BOARD",
		Version:     "1.0.0",
		Description: "Test Board",
		Chips:       []string{"TestChip"},
	}

	manifest.AddBoard(newBoard)

	if len(manifest.Boards) != 1 {
		t.Errorf("Expected 1 board, got %d", len(manifest.Boards))
	}
	if manifest.Boards[0].Name != "TEST-BOARD" {
		t.Errorf("Expected board name 'TEST-BOARD', got '%s'", manifest.Boards[0].Name)
	}
}

func TestAddApp(t *testing.T) {
	manifest := &Manifest{
		Name:    "test",
		Version: "1.0.0",
	}

	newApp := App{
		Name:        "test-app",
		Version:     "1.0.0",
		Description: "Test Application",
		Path:        "apps/test",
	}

	manifest.AddApp(newApp)

	if len(manifest.Apps) != 1 {
		t.Errorf("Expected 1 app, got %d", len(manifest.Apps))
	}
	if manifest.Apps[0].Name != "test-app" {
		t.Errorf("Expected app name 'test-app', got '%s'", manifest.Apps[0].Name)
	}
}

func TestParseInvalidXML(t *testing.T) {
	invalidXML := `<invalid>not a valid manifest</invalid>`
	reader := strings.NewReader(invalidXML)
	manifest, err := ParseManifest(reader)
	if err != nil {
		// This is expected - we just want to make sure it doesn't panic
		return
	}
	// If no error, at least the manifest should be somewhat empty
	if manifest.Name == "test-manifest" {
		t.Error("Should not have parsed invalid XML into expected manifest")
	}
}
