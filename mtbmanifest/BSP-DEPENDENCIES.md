# BSP Dependencies Manifest

## Overview

The BSP Dependencies Manifest defines the library and component dependencies for each BSP (Board Support Package) across different versions. This allows ModusToolbox to know which libraries to fetch when a user selects a specific board and version.

## URL
```
https://raw.githubusercontent.com/Infineon/mtb-bsp-manifest/v2.X/mtb-bsp-dependencies-manifest.xml
```

## Structure

### Root Element: `<dependencies>`
- **Attribute**: `version` - Manifest format version (e.g., "2.0")
- **Contains**: Multiple `<depender>` elements

### Depender
Represents a BSP (board) that depends on libraries/components.

- **`<id>`**: BSP identifier
  - Examples: `CY8CKIT-040T`, `PMG1-CY7110`, `KIT_XMC72_EVK_MUR_43439M2`
- **`<versions>`**: Container for version-specific dependencies
  - Contains multiple `<version>` elements

### Version
Represents dependencies for a specific BSP version.

- **`<commit>`**: Version/tag identifier
  - Examples: `latest-v3.X`, `release-v3.2.0`, `latest-v1.X`
- **`<dependees>`**: Container for dependencies
  - Contains multiple `<dependee>` elements

### Dependee
Represents a library or component that the BSP depends on.

- **`<id>`**: Library identifier
  - Common dependencies:
    - `core-lib` - Core library
    - `core-make` - Build system
    - `mtb-pdl-cat1` / `mtb-pdl-cat2` / `mtb-pdl-cat4` - Peripheral Driver Libraries
    - `mtb-hal-cat1` / `mtb-hal-cat2` / `mtb-hal-cat4` - Hardware Abstraction Layers
    - `recipe-make-cat1` / `recipe-make-cat2` / `recipe-make-cat4` - Build recipes
    - `capsense` - CapSense library
    - `cat1cm0p` - CM0+ core library
    - `bt-fw-mur-cyw43439` - Bluetooth firmware
- **`<commit>`**: Required version/tag of the dependency
  - Examples: `latest-v1.X`, `latest-v2.X`, `latest-v3.X`

## Usage Patterns

### 1. Get Dependencies for a BSP Version
When a user selects "CY8CKIT-040T" board with version "latest-v3.X", you need to know which libraries to fetch:

```go
deps, found := manifest.GetDependencies("CY8CKIT-040T", "latest-v3.X")
if found {
    for _, dep := range deps {
        // Fetch dep.ID at dep.Commit version
        fmt.Printf("Need to fetch: %s @ %s\n", dep.ID, dep.Commit)
    }
}
```

### 2. List Available Versions for a BSP
Show users what versions are available for their board:

```go
versions, found := manifest.GetBSPVersions("CY8CKIT-040T")
if found {
    fmt.Println("Available versions:")
    for _, v := range versions {
        fmt.Printf("  - %s\n", v)
    }
}
```

### 3. Find BSPs Using a Library
Determine which BSPs use a specific library (useful for impact analysis):

```go
bsps := manifest.FindBSPsUsingLibrary("mtb-pdl-cat2")
fmt.Printf("BSPs using mtb-pdl-cat2: %v\n", bsps)
```

### 4. Dependency Resolution
Build a complete dependency map for a BSP:

```go
depMap, err := resolveDependencies(&manifest, "PMG1-CY7110", "latest-v3.X")
if err == nil {
    for libID, version := range depMap {
        fmt.Printf("%s: %s\n", libID, version)
    }
}
```

## Version Patterns

### Semantic Versioning
- `latest-vX.X` - Tracks the latest release in that major version
  - `latest-v1.X` - Latest v1.x.x release
  - `latest-v2.X` - Latest v2.x.x release
  - `latest-v3.X` - Latest v3.x.x release

### Specific Releases
- `release-vX.Y.Z` - Exact release version
  - `release-v3.2.0`
  - `release-v1.1.0`

## Common Dependency Categories

### Category 1 (CAT1) - PSoC 6, XMC7000
Dependencies typically include:
- `core-lib`, `core-make`
- `mtb-pdl-cat1` - Peripheral drivers
- `mtb-hal-cat1` - Hardware abstraction
- `recipe-make-cat1` or `recipe-make-cat1c` - Build recipes
- `cat1cm0p` - CM0+ support (for dual-core)

### Category 2 (CAT2) - PSoC 4, PMG1
Dependencies typically include:
- `core-lib`, `core-make`
- `mtb-pdl-cat2` - Peripheral drivers
- `mtb-hal-cat2` - Hardware abstraction (older versions)
- `recipe-make-cat2` - Build recipes
- `capsense` - For touch-sensing boards

### Category 4 (CAT4) - CYW43xxx WiFi/BT
Dependencies typically include:
- `core-lib`, `core-make`
- `mtb-hal-cat4` - Hardware abstraction
- `recipe-make-cat4` - Build recipes

## Evolution Over Time

BSPs evolve through versions with changing dependencies:

**Example: CY8CKIT-041-41XX**

```
latest-v3.X (current)
  - No HAL dependency (direct PDL access)
  - mtb-pdl-cat2: latest-v2.X
  - core-make: latest-v3.X

latest-v2.X (previous)
  - Includes HAL
  - mtb-hal-cat2: latest-v2.X
  - mtb-pdl-cat2: latest-v1.X
  - core-make: latest-v1.X
  - capsense: latest-v3.X

latest-v1.X (older)
  - capsense: latest-v2.X
  - mtb-hal-cat2: latest-v1.X
```

This shows the progression:
1. Older versions used HAL + older libraries
2. v2 updated to newer libraries, kept HAL
3. v3 removed HAL dependency, uses direct PDL access

## Integration with Other Manifests

This manifest works in concert with:
1. **BSP Manifest** - Defines what BSPs exist and where to find them
2. **Middleware Manifest** - Defines the libraries referenced in `<dependee>` elements
3. **Super Manifest** - Points to this manifest as a child

Workflow:
1. User selects BSP from BSP Manifest
2. Look up dependencies in BSP Dependencies Manifest
3. Fetch each dependency from URLs in Middleware Manifest

## MCP Server Use Cases

### Query: "What libraries does the CY8CKIT-040T board need?"
```
Answer: For latest-v3.X:
- core-lib (latest-v1.X)
- core-make (latest-v3.X)
- mtb-pdl-cat2 (latest-v2.X)
- recipe-make-cat2 (latest-v2.X)
```

### Query: "Which boards use the capsense library?"
```
Answer: 
- CY8CKIT-041-41XX
- CY8CKIT-041S-MAX
- CY8CKIT-045S
- CY8CKIT-145-40XX
- CY8CKIT-149
- PMG1-CY7113
```

### Query: "What changed between v2.X and v3.X of CY8CKIT-041-41XX?"
```
Answer:
Removed dependencies:
- capsense
- mtb-hal-cat2

Updated dependencies:
- core-make: v1.X → v3.X
- mtb-pdl-cat2: v1.X → v2.X
- recipe-make-cat2: v1.X → v2.X
```

## Data Statistics

From the current manifest:
- **Total BSPs**: ~15+ boards
- **Versions per BSP**: 3-15 versions
- **Dependencies per version**: 4-7 dependencies
- **Common libraries**: ~20 unique library IDs
- **Version schemes**: `latest-vX.X`, `release-vX.Y.Z`

## Notes

- Not all BSPs have the same number of versions
- Newer BSPs may have fewer historical versions
- Dependencies can be added/removed between versions
- The "latest-vX.X" versions track rolling releases
- Some BSPs have special dependencies (e.g., BT firmware)
