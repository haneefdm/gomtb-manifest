# gomtb-manifest

A Go library for parsing and manipulating ModusToolbox (MTB) manifest files.

## Overview

`gomtb-manifest` provides utilities for working with ModusToolbox manifest files, which are XML-based configuration files used to define dependencies, board support packages, and applications in ModusToolbox projects.

## Features

- **Parse MTB manifest files** from XML format
- **Create and modify** manifest structures programmatically
- **Validate** manifest integrity and structure
- **Query and filter** dependencies, boards, and applications
- **Export** manifests back to XML format
- **Comprehensive utility functions** for common operations

## Installation

```bash
go get github.com/haneefdm/gomtb-manifest
```

## Quick Start

### Parsing a Manifest File

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/haneefdm/gomtb-manifest"
)

func main() {
    // Parse from file
    manifest, err := gomtb.ParseManifestFile("manifest.xml")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Manifest: %s v%s\n", manifest.Name, manifest.Version)
    fmt.Printf("Dependencies: %d\n", len(manifest.Dependencies))
}
```

### Creating a Manifest

```go
manifest := &gomtb.Manifest{
    Name:        "my-project",
    Version:     "1.0.0",
    Description: "My ModusToolbox Project",
}

// Add a dependency
manifest.AddDependency(gomtb.Dependency{
    Name:     "core-lib",
    Version:  "1.2.3",
    URI:      "https://github.com/example/core-lib",
    Required: true,
})

// Add a board
manifest.AddBoard(gomtb.Board{
    Name:        "CY8CPROTO-062-4343W",
    Version:     "2.5.0",
    Description: "PSoC 6 Wi-Fi BT Prototyping Kit",
    Chips:       []string{"PSoC6", "CYW4343W"},
})

// Save to file
err := manifest.WriteToFile("manifest.xml")
```

### Querying Dependencies

```go
// Get a specific dependency
dep := manifest.GetDependency("core-lib")
if dep != nil {
    fmt.Printf("Version: %s\n", dep.Version)
}

// Get all required dependencies
required := manifest.GetRequiredDependencies()

// Get all optional dependencies
optional := manifest.GetOptionalDependencies()

// Custom filtering
filtered := manifest.FilterDependencies(func(dep gomtb.Dependency) bool {
    return dep.Version == "1.0.0"
})
```

### Querying Boards

```go
// Get a specific board
board := manifest.GetBoard("CY8CPROTO-062-4343W")

// Get boards by chip type
psoc6Boards := manifest.GetBoardsByChip("PSoC6")

// Custom filtering
filtered := manifest.FilterBoards(func(board gomtb.Board) bool {
    return len(board.Chips) > 1
})
```

### Validation

```go
errors := gomtb.ValidateManifest(manifest)
if len(errors) > 0 {
    for _, err := range errors {
        fmt.Printf("Validation error: %v\n", err)
    }
}
```

### Getting a Summary

```go
summary := manifest.Summary()
fmt.Print(summary)
// Output:
// Manifest: my-project (v1.0.0)
// Description: My ModusToolbox Project
// Dependencies: 3
// Boards: 2
// Apps: 1
```

## API Documentation

### Core Types

#### `Manifest`
The main structure representing an MTB manifest file.

**Fields:**
- `Name` (string) - Manifest name
- `Version` (string) - Manifest version
- `Description` (string) - Manifest description
- `Dependencies` ([]Dependency) - List of dependencies
- `Boards` ([]Board) - List of board support packages
- `Apps` ([]App) - List of applications

#### `Dependency`
Represents a dependency entry.

**Fields:**
- `Name` (string) - Dependency name
- `Version` (string) - Dependency version
- `URI` (string) - Repository URI
- `Required` (bool) - Whether the dependency is required

#### `Board`
Represents a board support package.

**Fields:**
- `Name` (string) - Board name
- `Version` (string) - Board version
- `Description` (string) - Board description
- `Chips` ([]string) - List of supported chips

#### `App`
Represents an application entry.

**Fields:**
- `Name` (string) - Application name
- `Version` (string) - Application version
- `Description` (string) - Application description
- `Path` (string) - Application path

### Functions

#### Parsing Functions
- `ParseManifest(r io.Reader) (*Manifest, error)` - Parse manifest from a reader
- `ParseManifestFile(path string) (*Manifest, error)` - Parse manifest from a file

#### Export Functions
- `ToXML() ([]byte, error)` - Convert manifest to XML
- `WriteToFile(path string) error` - Write manifest to a file

#### Query Functions
- `GetDependency(name string) *Dependency` - Get dependency by name
- `GetBoard(name string) *Board` - Get board by name
- `GetApp(name string) *App` - Get app by name
- `HasDependency(name string) bool` - Check if dependency exists
- `HasBoard(name string) bool` - Check if board exists
- `HasApp(name string) bool` - Check if app exists

#### Filter Functions
- `FilterDependencies(filter func(Dependency) bool) []Dependency` - Filter dependencies
- `GetRequiredDependencies() []Dependency` - Get required dependencies
- `GetOptionalDependencies() []Dependency` - Get optional dependencies
- `FilterBoards(filter func(Board) bool) []Board` - Filter boards
- `GetBoardsByChip(chipName string) []Board` - Get boards by chip
- `FilterApps(filter func(App) bool) []App` - Filter apps

#### Modification Functions
- `AddDependency(dep Dependency)` - Add a dependency
- `AddBoard(board Board)` - Add a board
- `AddApp(app App)` - Add an app
- `RemoveDependency(name string) bool` - Remove a dependency
- `RemoveBoard(name string) bool` - Remove a board
- `RemoveApp(name string) bool` - Remove an app

#### Utility Functions
- `ValidateManifest(m *Manifest) []error` - Validate manifest structure
- `Summary() string` - Get human-readable summary

## Example Manifest Format

```xml
<?xml version="1.0" encoding="UTF-8"?>
<manifest name="my-project" version="1.0.0">
  <description>My ModusToolbox Project</description>
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
</manifest>
```

## Testing

Run the test suite:

```bash
go test -v
```

Run tests with coverage:

```bash
go test -cover
```

## License

See LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.