package mtbmanifest

import (
	"fmt"
	"os"
)

// Example usage of CE manifest parsing

func CEExampleMain() {
	// Example 1: Parse a v1 manifest
	fmt.Println("=== Example 1: Parsing v1 manifest ===")
	v1Data := `<apps>
  <app>
    <n>Empty App</n>
    <id>mtb-example-empty-app-0</id>
    <uri>https://github.com/Infineon/mtb-example-empty-app</uri>
    <description><![CDATA[This empty application provides a template]]></description>
    <req_capabilities>psoc6</req_capabilities>
    <versions>
      <version req_capabilities_per_version="bsp_gen1" tools_max_version="2.1.0">
        <num>Latest 1.X release</num>
        <commit>latest-v1.X</commit>
      </version>
    </versions>
  </app>
</apps>`

	var apps1 Apps
	if err := UnmarshalXMLWithVerification([]byte(v1Data), &apps1); err != nil {
		fmt.Printf("Error parsing v1: %v\n", err)
		return
	}

	fmt.Printf("Manifest version: v%s (v2=%v)\n", apps1.Version, apps1.IsV2())
	fmt.Printf("Number of apps: %d\n", len(apps1.App))

	app := apps1.App[0]
	fmt.Printf("\nApp: %s\n", app.Name)
	fmt.Printf("ID: %s\n", app.ID)

	caps := app.GetCapabilities()
	fmt.Printf("Capabilities: %s\n", caps.String())
	fmt.Printf("Is v2 format: %v\n", caps.IsV2)

	// Example 2: Parse a v2 manifest with complex capabilities
	fmt.Println("\n=== Example 2: Parsing v2 manifest with complex capabilities ===")
	v2Data := `<apps version="2.0">
  <app keywords="led,starter,hello world,mtb-flow" req_capabilities_v2="hal led [psoc6,t2gbe,xmc7000] [flash_0k,flash_2048k,flash_1024k]">
    <n>Hello World</n>
    <id>mtb-example-hal-hello-world</id>
    <category>Getting Started</category>
    <uri>https://github.com/Infineon/mtb-example-hal-hello-world</uri>
    <description><![CDATA[This code example demonstrates simple UART communication]]></description>
    <versions>
      <version flow_version="2.0" tools_min_version="3.1.0" req_capabilities_per_version_v2="[bsp_gen5,bsp_gen4]">
        <num>Latest 4.X release</num>
        <commit>latest-v4.X</commit>
      </version>
    </versions>
  </app>
</apps>`

	var apps2 Apps
	if err := UnmarshalXMLWithVerification([]byte(v2Data), &apps2); err != nil {
		fmt.Printf("Error parsing v2: %v\n", err)
		return
	}

	fmt.Printf("Manifest version: v%s (v2=%v)\n", apps2.Version, apps2.IsV2())

	app2 := apps2.App[0]
	fmt.Printf("\nApp: %s\n", app2.Name)
	fmt.Printf("Category: %s\n", app2.Category)
	fmt.Printf("Keywords: %v\n", app2.GetKeywords())

	caps2 := app2.GetCapabilities()
	fmt.Printf("\nCapabilities: %s\n", caps2.String())
	fmt.Printf("Is v2 format: %v\n", caps2.IsV2)
	fmt.Printf("Capability groups breakdown:\n")
	for i, group := range caps2.Groups {
		fmt.Printf("  Group %d: %v\n", i+1, group)
	}

	// Example 3: Check if capabilities match
	fmt.Println("\n=== Example 3: Capability matching ===")

	// Test scenario 1: Board with PSoC6, 2MB flash, and HAL+LED
	available1 := map[string]bool{
		"psoc6":       true,
		"hal":         true,
		"led":         true,
		"flash_2048k": true,
	}

	matches1 := caps2.Matches(available1)
	fmt.Printf("Board 1 (PSoC6, 2MB flash, HAL, LED): %v\n", matches1)

	// Test scenario 2: Board with XMC7000, 1MB flash, HAL+LED
	available2 := map[string]bool{
		"xmc7000":     true,
		"hal":         true,
		"led":         true,
		"flash_1024k": true,
	}

	matches2 := caps2.Matches(available2)
	fmt.Printf("Board 2 (XMC7000, 1MB flash, HAL, LED): %v\n", matches2)

	// Test scenario 3: Board missing LED capability
	available3 := map[string]bool{
		"psoc6":       true,
		"hal":         true,
		"flash_2048k": true,
	}

	matches3 := caps2.Matches(available3)
	fmt.Printf("Board 3 (PSoC6, 2MB flash, HAL, no LED): %v\n", matches3)

	// Example 4: Version-specific capabilities
	fmt.Println("\n=== Example 4: Version-specific capabilities ===")

	version := app2.Versions.Version[0]
	fmt.Printf("Version: %s\n", version.Num)

	toolsVer, isMin := version.GetToolsVersion()
	if isMin {
		fmt.Printf("Minimum tools version: %s\n", toolsVer)
	} else {
		fmt.Printf("Maximum tools version: %s\n", toolsVer)
	}

	versionCaps := version.GetCapabilities()
	fmt.Printf("Version capabilities: %s\n", versionCaps.String())

	// Example 5: Parsing various capability formats
	fmt.Println("\n=== Example 5: Parsing various capability formats ===")

	testCases := []string{
		"psoc6 led",             // v1: simple AND
		"[psoc6,t2gbe]",         // v2: OR group only
		"hal [psoc6,t2gbe] led", // v2: mixed
		"[cat1a,cat1b] [flash_2048k,flash_1024k] hal", // v2: multiple OR groups
		"", // empty
	}

	for _, tc := range testCases {
		parsed := ParseCapabilities(tc)
		fmt.Printf("Input:  %q\n", tc)
		fmt.Printf("Parsed: %s\n", parsed.String())
		fmt.Printf("Format: v%d\n\n", map[bool]int{false: 1, true: 2}[parsed.IsV2])
	}
}

// LoadManifestFromFile demonstrates loading from actual files
func LoadManifestFromFile(filename string) (*Apps, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var apps Apps
	if err := UnmarshalXMLWithVerification(data, &apps); err != nil {
		return nil, fmt.Errorf("parse XML: %w", err)
	}

	return &apps, nil
}

// FindCompatibleApps returns apps that match the given capabilities
func FindCompatibleApps(apps *Apps, availableCapabilities map[string]bool) []*App {
	compatible := make([]*App, 0)

	for _, app := range apps.App {
		caps := app.GetCapabilities()
		if caps.Matches(availableCapabilities) {
			compatible = append(compatible, app)
		}
	}

	return compatible
}

// Example of finding compatible versions for a specific app
func FindCompatibleVersions(app *App, availableCapabilities map[string]bool) []*CEVersion {
	compatible := make([]*CEVersion, 0)

	for _, version := range app.Versions.Version {
		versionCaps := version.GetCapabilities()
		if versionCaps.Matches(availableCapabilities) {
			compatible = append(compatible, version)
		}
	}

	return compatible
}
