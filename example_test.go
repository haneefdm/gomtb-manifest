package gomtb_test

import (
	"fmt"
	"log"
	"strings"

	"github.com/haneefdm/gomtb-manifest"
)

func ExampleParseManifest() {
	manifestXML := `<?xml version="1.0" encoding="UTF-8"?>
<manifest name="my-project" version="1.0.0">
  <description>My ModusToolbox Project</description>
  <dependencies>
    <dependency name="core-lib" version="1.2.3" uri="https://github.com/example/core-lib" required="true"/>
  </dependencies>
</manifest>`

	manifest, err := gomtb.ParseManifest(strings.NewReader(manifestXML))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Name: %s\n", manifest.Name)
	fmt.Printf("Version: %s\n", manifest.Version)
	fmt.Printf("Dependencies: %d\n", len(manifest.Dependencies))
	// Output:
	// Name: my-project
	// Version: 1.0.0
	// Dependencies: 1
}

func ExampleManifest_GetDependency() {
	manifest := &gomtb.Manifest{
		Name:    "my-project",
		Version: "1.0.0",
		Dependencies: []gomtb.Dependency{
			{Name: "core-lib", Version: "1.2.3", Required: true},
			{Name: "helper-lib", Version: "2.0.0", Required: false},
		},
	}

	dep := manifest.GetDependency("core-lib")
	if dep != nil {
		fmt.Printf("Found: %s v%s (required: %v)\n", dep.Name, dep.Version, dep.Required)
	}
	// Output:
	// Found: core-lib v1.2.3 (required: true)
}

func ExampleManifest_GetRequiredDependencies() {
	manifest := &gomtb.Manifest{
		Dependencies: []gomtb.Dependency{
			{Name: "core-lib", Version: "1.2.3", Required: true},
			{Name: "helper-lib", Version: "2.0.0", Required: false},
			{Name: "essential-lib", Version: "1.0.0", Required: true},
		},
	}

	required := manifest.GetRequiredDependencies()
	fmt.Printf("Required dependencies: %d\n", len(required))
	// Output:
	// Required dependencies: 2
}

func ExampleManifest_GetBoardsByChip() {
	manifest := &gomtb.Manifest{
		Boards: []gomtb.Board{
			{Name: "CY8CPROTO-062-4343W", Chips: []string{"PSoC6", "CYW4343W"}},
			{Name: "CY8CKIT-062-BLE", Chips: []string{"PSoC6", "BLE"}},
			{Name: "CY8CKIT-041-41XX", Chips: []string{"PSoC4"}},
		},
	}

	psoc6Boards := manifest.GetBoardsByChip("PSoC6")
	fmt.Printf("Boards with PSoC6: %d\n", len(psoc6Boards))
	// Output:
	// Boards with PSoC6: 2
}

func ExampleManifest_Summary() {
	manifest := &gomtb.Manifest{
		Name:         "my-project",
		Version:      "1.0.0",
		Description:  "My ModusToolbox Project",
		Dependencies: []gomtb.Dependency{{Name: "core-lib"}},
		Boards:       []gomtb.Board{{Name: "board1"}},
		Apps:         []gomtb.App{{Name: "app1"}},
	}

	fmt.Print(manifest.Summary())
	// Output:
	// Manifest: my-project (v1.0.0)
	// Description: My ModusToolbox Project
	// Dependencies: 1
	// Boards: 1
	// Apps: 1
}

func ExampleValidateManifest() {
	manifest := &gomtb.Manifest{
		Name:    "my-project",
		Version: "1.0.0",
		Dependencies: []gomtb.Dependency{
			{Name: "core-lib", Version: "1.2.3"},
			{Name: "core-lib", Version: "2.0.0"}, // Duplicate!
		},
	}

	errors := gomtb.ValidateManifest(manifest)
	fmt.Printf("Validation errors: %d\n", len(errors))
	// Output:
	// Validation errors: 1
}

func ExampleManifest_AddDependency() {
	manifest := &gomtb.Manifest{
		Name:    "my-project",
		Version: "1.0.0",
	}

	manifest.AddDependency(gomtb.Dependency{
		Name:     "new-lib",
		Version:  "1.0.0",
		URI:      "https://github.com/example/new-lib",
		Required: true,
	})

	fmt.Printf("Dependencies: %d\n", len(manifest.Dependencies))
	// Output:
	// Dependencies: 1
}

func ExampleManifest_RemoveDependency() {
	manifest := &gomtb.Manifest{
		Dependencies: []gomtb.Dependency{
			{Name: "dep1", Version: "1.0.0"},
			{Name: "dep2", Version: "2.0.0"},
		},
	}

	removed := manifest.RemoveDependency("dep1")
	fmt.Printf("Removed: %v, Remaining: %d\n", removed, len(manifest.Dependencies))
	// Output:
	// Removed: true, Remaining: 1
}
