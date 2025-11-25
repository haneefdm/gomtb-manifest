# Code Example Manifest Parser

## Overview

This Go package provides comprehensive support for parsing Infineon ModusToolbox Code Example (CE) manifests in both v1 and v2 formats.

**Manifest URLs:**
- v1: https://github.com/Infineon/mtb-ce-manifest/raw/v2.X/mtb-ce-manifest.xml
- v2: https://github.com/Infineon/mtb-ce-manifest/raw/v2.X/mtb-ce-manifest-fv2.xml

## Files

1. **ce_manifest.go** - Core structs and helper functions
2. **ce_manifest_example.go** - Usage examples
3. **ce_manifest_test.go** - Comprehensive test suite

## Key Structures

### Apps
Root element containing all code examples:
```go
type Apps struct {
    XMLName xml.Name `xml:"apps"`
    Version string   `xml:"version,attr,omitempty"` // "2.0" for v2
    App     []App    `xml:"app"`
}
```

### App
Individual code example:
```go
type App struct {
    XMLName           xml.Name   `xml:"app"`
    Keywords          string     `xml:"keywords,attr,omitempty"`          // v2 only
    ReqCapabilities   string     `xml:"req_capabilities,attr,omitempty"`  // v1
    ReqCapabilitiesV2 string     `xml:"req_capabilities_v2,attr,omitempty"` // v2
    Name              string     `xml:"n"`
    ID                string     `xml:"id"`
    Category          string     `xml:"category,omitempty"` // v2 only
    URI               string     `xml:"uri"`
    Description       string     `xml:"description"`
    Versions          CEVersions `xml:"versions"`
}
```

### CEVersion
Version-specific information:
```go
type CEVersion struct {
    XMLName                     xml.Name `xml:"version"`
    FlowVersion                 string   `xml:"flow_version,attr,omitempty"`
    ToolsMinVersion             string   `xml:"tools_min_version,attr,omitempty"` // v2
    ToolsMaxVersion             string   `xml:"tools_max_version,attr,omitempty"` // v1
    ReqCapabilitiesPerVersion   string   `xml:"req_capabilities_per_version,attr,omitempty"` // v1
    ReqCapabilitiesPerVersionV2 string   `xml:"req_capabilities_per_version_v2,attr,omitempty"` // v2
    Num                         string   `xml:"num"`
    Commit                      string   `xml:"commit"`
}
```

## Capability Parsing

### Format Differences

**v1 Format (space-delimited):**
```
"psoc6 led capsense_button"
```
All capabilities are required (implicit AND).

**v2 Format (bracketed OR groups):**
```
"hal led [psoc6,t2gbe,xmc7000] [flash_2048k,flash_1024k]"
```
- Items in `[brackets,separated,by,commas]` are OR'd together
- Multiple groups and plain items are AND'd together
- Above means: `hal AND led AND (psoc6 OR t2gbe OR xmc7000) AND (flash_2048k OR flash_1024k)`

### CapabilityRequirement

Parsed capability structure:
```go
type CapabilityRequirement struct {
    Groups [][]string // Each group is OR'd internally, groups are AND'd together
    IsV2   bool       // True if parsed from v2 bracketed syntax
}
```

## Helper Functions

### ParseCapabilities
```go
func ParseCapabilities(capString string) CapabilityRequirement
```
Automatically detects and parses either v1 or v2 format.

**Examples:**
```go
// v1 format
caps := ParseCapabilities("psoc6 led")
// Result: 2 groups: [[psoc6], [led]]
// Logic: psoc6 AND led

// v2 format
caps := ParseCapabilities("[psoc6,t2gbe] hal led")
// Result: 3 groups: [[psoc6,t2gbe], [hal], [led]]
// Logic: (psoc6 OR t2gbe) AND hal AND led
```

### App.GetCapabilities
```go
func (a *App) GetCapabilities() CapabilityRequirement
```
Returns parsed capabilities, preferring v2 format if available.

### CEVersion.GetCapabilities
```go
func (v *CEVersion) GetCapabilities() CapabilityRequirement
```
Returns parsed version-specific capabilities.

### CapabilityRequirement.Matches
```go
func (cr *CapabilityRequirement) Matches(availableCaps map[string]bool) bool
```
Checks if available capabilities satisfy the requirement.

**Example:**
```go
caps := ParseCapabilities("[psoc6,t2gbe] hal led [flash_2048k,flash_1024k]")

// Board 1: PSoC6 with 2MB flash, HAL, LED
available1 := map[string]bool{
    "psoc6": true,
    "hal": true,
    "led": true,
    "flash_2048k": true,
}
matches1 := caps.Matches(available1) // true

// Board 2: XMC7000 with 1MB flash, HAL, LED
available2 := map[string]bool{
    "xmc7000": true,  // Doesn't match [psoc6,t2gbe]
    "hal": true,
    "led": true,
    "flash_1024k": true,
}
matches2 := caps.Matches(available2) // false - missing psoc6/t2gbe group
```

### App.GetKeywords
```go
func (a *App) GetKeywords() []string
```
Parses comma-delimited keywords into a slice (v2 only).

### CEVersion.GetToolsVersion
```go
func (v *CEVersion) GetToolsVersion() (version string, isMin bool)
```
Returns the appropriate tools version (min for v2, max for v1).

### Apps.IsV2
```go
func (apps *Apps) IsV2() bool
```
Returns true if this is a v2 format manifest (version="2.0").

## Usage Examples

### Basic Parsing

```go
// Load and parse manifest
data, _ := os.ReadFile("mtb-ce-manifest-fv2.xml")
var apps Apps
xml.Unmarshal(data, &apps)

// Iterate through code examples
for _, app := range apps.App {
    fmt.Printf("App: %s\n", app.Name)
    fmt.Printf("Category: %s\n", app.Category)
    
    caps := app.GetCapabilities()
    fmt.Printf("Requirements: %s\n", caps.String())
    
    // Check each version
    for _, version := range app.Versions.Version {
        fmt.Printf("  Version: %s\n", version.Num)
        fmt.Printf("  Commit: %s\n", version.Commit)
    }
}
```

### Finding Compatible Apps

```go
// Define available hardware capabilities
boardCapabilities := map[string]bool{
    "psoc6":       true,
    "hal":         true,
    "led":         true,
    "flash_2048k": true,
    "bsp_gen4":    true,
}

// Find compatible apps
compatible := []App{}
for _, app := range apps.App {
    appCaps := app.GetCapabilities()
    if appCaps.Matches(boardCapabilities) {
        // Check if any version is compatible
        for _, version := range app.Versions.Version {
            versionCaps := version.GetCapabilities()
            if versionCaps.Matches(boardCapabilities) {
                compatible = append(compatible, app)
                break
            }
        }
    }
}

fmt.Printf("Found %d compatible apps\n", len(compatible))
```

### Comparing v1 and v2

```go
// Check manifest version
if apps.IsV2() {
    fmt.Println("Using v2 manifest format")
    
    // v2-specific features
    for _, app := range apps.App {
        keywords := app.GetKeywords()
        fmt.Printf("%s: Keywords=%v, Category=%s\n", 
            app.Name, keywords, app.Category)
    }
} else {
    fmt.Println("Using v1 manifest format")
}
```

## Capability Format Examples

### Simple Requirements
```
v1: "psoc6 led"
v2: "psoc6 led"
Meaning: Requires psoc6 AND led
```

### OR Logic (v2 only)
```
v2: "[psoc6,t2gbe,xmc7000]"
Meaning: Requires (psoc6 OR t2gbe OR xmc7000)
```

### Complex Boolean Logic (v2)
```
v2: "hal [psoc6,t2gbe] [flash_2048k,flash_1024k,flash_512k]"
Meaning: Requires hal AND (psoc6 OR t2gbe) AND (flash_2048k OR flash_1024k OR flash_512k)
```

### Real-World Example
```
v2: "hal led [psoc6,t2gbe,xmc7000,kit_t2g_b_h_evk] [flash_0k,flash_2048k,flash_1024k]"

This means the app requires:
1. hal (required)
2. led (required)
3. At least one of: psoc6, t2gbe, xmc7000, or kit_t2g_b_h_evk
4. At least one of: flash_0k, flash_2048k, or flash_1024k
```

## Key Differences: v1 vs v2

| Feature | v1 | v2 |
|---------|----|----|
| Root element | `<apps>` | `<apps version="2.0">` |
| Capabilities attribute | `req_capabilities` | `req_capabilities_v2` |
| Capability format | Space-delimited | Bracketed OR groups |
| Keywords | Not present | `keywords` attribute |
| Category | Not present | `<category>` element |
| Tools version | `tools_max_version` | `tools_min_version` |
| Per-version caps | `req_capabilities_per_version` | `req_capabilities_per_version_v2` |

## Testing

Run the test suite:
```bash
go test -v ce_manifest_test.go ce_manifest.go
```

The test suite covers:
- v1 capability parsing
- v2 capability parsing (simple and complex)
- Capability matching logic
- XML parsing for both formats
- Keyword parsing
- Tool version handling
- Edge cases (empty strings, single capabilities, etc.)

## Integration with Other Manifests

This CE manifest parser complements the other manifest parsers you have:

1. **BSP Manifest** - Board definitions
2. **Middleware Manifest** - Available libraries
3. **Capabilities Manifest** - Capability token descriptions
4. **Dependencies Manifest** - Version requirements
5. **CE Manifest** (this one) - Code examples and their requirements

Together, these allow you to:
- Find boards that support a code example
- Determine which libraries are needed
- Verify version compatibility
- Understand capability requirements
- 
