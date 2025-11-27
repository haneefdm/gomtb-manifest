# BSP Capabilities Manifest

## Overview

The BSP Capabilities Manifest is the **"dictionary"** that defines what all the capability tokens in BSP manifests mean. When a BSP lists capabilities like `"ble"`, `"flash_256k"`, or `"capsense_button"`, this manifest explains what each token represents.

## URL
```
https://raw.githubusercontent.com/Infineon/mtb-bsp-manifest/v2.X/mtb-bsp-capabilities-manifest.json
```

## Structure

### Root Element
```json
{
  "capabilities": [ ... ]
}
```

### Capability
Each capability has five fields:

```json
{
  "category": "Memory",
  "description": "This chip has 256K of flash memory.",
  "name": "flash_256k",
  "token": "flash_256k",
  "types": ["chip"]
}
```

**Fields:**
- **`category`**: Groups related capabilities (e.g., "Memory", "Networking", "Chip Families")
- **`description`**: Human-readable explanation of what this capability means
- **`name`**: Display name for the capability
- **`token`**: Unique identifier used in BSP manifest XML files
- **`types`**: Where this capability applies - `["chip"]`, `["board"]`, or `["generation"]`

## Capability Categories

The manifest organizes capabilities into these categories:

### 1. Chip Families
Identifies which chip family/series a device belongs to.

Examples:
- `cat1`, `cat1a`, `cat1b`, `cat1c` - PSoC 6, XMC7000, etc.
- `cat2` - PSoC 4
- `cat3` - XMC1000
- `cat4` - CYW43xxx WiFi/BT chips
- `cat5` - CYW55xxx chips
- `psoc4`, `psoc6` - Specific PSoC families
- `xmc`, `xmc1000`, `xmc4000`, `xmc7000` - XMC families
- `pmg1` - USB-C power delivery chips
- `cyw20819`, `cyw43012`, `cyw55513` - Specific chip models

### 2. Memory
Specifies memory capacity.

**Flash Memory:**
- Pattern: `flash_XYZk` where XYZ is the size in KB
- Examples: `flash_256k`, `flash_512k`, `flash_1024k`, `flash_2048k`
- Range: 8K to 8384K

**SRAM:**
- Pattern: `sram_XYZk` where XYZ is the size in KB
- Examples: `sram_128k`, `sram_256k`, `sram_512k`, `sram_1024k`
- Range: 2K to 2048K

**Other:**
- `fram` - FRAM memory present
- `nor_flash` - NOR flash present
- `memory`, `memory_i2c`, `memory_qspi` - External memory

### 3. Networking
Communication protocols and wireless capabilities.

Examples:
- `wifi` - WiFi support
- `ble` - Bluetooth Low Energy
- `bt` - Bluetooth (general)
- `br_edr` - Bluetooth Classic (Basic Rate / Enhanced Data Rate)
- `ethernet` - Ethernet support
- `mesh` - Mesh networking over Bluetooth
- `audio` - Audio over Bluetooth
- `ota` - Over-The-Air firmware updates

### 4. Hardware Blocks
On-chip peripherals and hardware modules.

Examples:
- `adc` - Analog to Digital Converter
- `dac` - Digital to Analog Converter
- `i2c` - I2C interface
- `spi` - SPI interface
- `uart` - UART
- `can` - Controller Area Network
- `usb_device`, `usb_host` - USB support
- `qspi` - Quad SPI
- `dma` - Direct Memory Access
- `rtc` - Real Time Clock
- `comp` - Analog comparator
- `opamp` - Operational amplifier

### 5. Human Interface Devices
User interaction components on boards.

Examples:
- `led` - At least one LED
- `led2x` - Two or more LEDs
- `rgb_led` - RGB LED
- `button` - Physical button
- `switch` - Switch/button
- `pot` - Potentiometer
- `capsense_button`, `capsense_slider`, `capsense_touchpad` - CapSense elements
- `msc_button`, `msc_slider` - MultiSense elements

### 6. Sensors
External sensor chips on boards.

Examples:
- `MAX44009EDT` - Ambient light sensor
- `lsm9ds1` - Motion sensor
- `ncp15xv103`, `ncu15wf104` - Temperature sensors

### 7. BSP Generation
Indicates which generation of BSP this belongs to.

Examples:
- `bsp_gen1` through `bsp_gen6` - BSP generation markers
- Later generations have improved features and tooling support

### 8. Software/Firmware Support
Software stack compatibility.

Examples:
- `hal` - Hardware Abstraction Layer support
- `btsdk` - Bluetooth SDK support
- `btstack10`, `btstack30` - Bluetooth stack versions
- `fw2` - Firmware version 2

### 9. Ports
Physical connector compatibility.

Examples:
- `arduino` - Arduino shield compatible
- `j2` - Arduino J2 header
- `feather` - Adafruit Feather compatible

### 10. Authentication/Security
Security features.

Examples:
- `optiga_trust_b`, `optiga_trust_m` - OPTIGA security chips
- `secure_boot` - Secure boot capability
- `std_crypto` - Standard crypto APIs
- `epc2`, `epc3`, `epc4` - Security families

### 11. Miscellaneous
Other capabilities.

Examples:
- `low_power` - Low-power mode support
- `capsense`, `csd` - CapSense sensing
- `smart_io` - Smart I/O pins
- `enclosure` - Board is in sealed enclosure
- `buck_converter` - Digital buck converter

## Capability Types

### `"chip"` Type
Inherent properties of the silicon chip itself.

Examples:
- Memory sizes: `flash_256k`, `sram_128k`
- Chip families: `psoc6`, `cat1a`
- Hardware blocks: `adc`, `i2c`, `usb_device`
- Networking: `wifi`, `ble`, `ethernet`

### `"board"` Type
Features of the development board/kit.

Examples:
- Components: `led`, `button`, `rgb_led`
- Sensors: `MAX44009EDT`, `lsm9ds1`
- External memory: `memory_qspi`, `fram`
- Interfaces: `arduino`, `feather`

### `"generation"` Type
BSP tooling generation markers.

Examples:
- `bsp_gen1` through `bsp_gen6`
- Indicates compatibility with ModusToolbox versions

## Usage Patterns

### 1. Explain Capability Tokens
When you encounter tokens in a BSP manifest:

```go
// BSP has these capability tokens
tokens := []string{"psoc6", "wifi", "ble", "flash_2048k", "led"}

// Look up what they mean
explanations := capManifest.ExplainTokens(tokens)
for token, description := range explanations {
    fmt.Printf("%s: %s\n", token, description)
}
```

Output:
```
psoc6: This chip is in the PSOC 6 family.
wifi: This chip supports WiFi.
ble: This chip supports Bluetooth Low Energy networking.
flash_2048k: This chip has 2048K of flash memory.
led: This board contains at least one user-controllable LED.
```

### 2. Filter BSPs by Capability
Find all boards with specific features:

```go
// Find all BSPs with WiFi
for _, bsp := range allBSPs {
    for _, cap := range bsp.Capabilities {
        if cap == "wifi" {
            fmt.Printf("Board with WiFi: %s\n", bsp.Name)
        }
    }
}
```

### 3. Search for Capabilities
Find capabilities matching a query:

```go
// User asks: "Which boards support bluetooth?"
results := capManifest.SearchCapabilities("bluetooth")
// Returns: ble, bt, br_edr, audio, mesh, etc.
```

### 4. Validate Capability Tokens
Check if a token is valid:

```go
if !capManifest.ValidateToken("unknown_token") {
    fmt.Println("Warning: Unknown capability token")
}
```

### 5. Browse by Category
Show all capabilities in a category:

```go
memoryCaps := capManifest.GetCapabilitiesByCategory("Memory")
for _, cap := range memoryCaps {
    fmt.Printf("- %s: %s\n", cap.Token, cap.Description)
}
```

## Integration with BSP Manifest

The BSP Manifest references capability tokens:

```xml
<board>
  <id>CY8CPROTO-062-4343W</id>
  <capabilities>
    <capability>psoc6</capability>
    <capability>wifi</capability>
    <capability>ble</capability>
    <capability>flash_2048k</capability>
    <capability>sram_1024k</capability>
  </capabilities>
</board>
```

The Capabilities Manifest explains what each token means, so an MCP server can answer:

**Query:** "Tell me about the CY8CPROTO-062-4343W board"

**Answer:** "This board has a PSoC 6 chip with 2MB flash and 1MB SRAM. It supports WiFi and Bluetooth Low Energy."

## MCP Server Use Cases

### Query: "Show me boards with WiFi and BLE"
```
1. Look up "wifi" token → "This chip supports WiFi"
2. Look up "ble" token → "This chip supports Bluetooth Low Energy"
3. Filter BSPs that have both tokens in their capabilities
```

### Query: "What memory does the CY8CKIT-062-BLE have?"
```
1. Get BSP capabilities: ["flash_1024k", "sram_288k", ...]
2. Look up memory tokens in capabilities manifest
3. Answer: "1024K flash, 288K SRAM"
```

### Query: "Which boards have CapSense buttons?"
```
1. Search capabilities for "capsense"
2. Find: capsense_button, capsense_slider, capsense_touchpad, etc.
3. Filter BSPs with "capsense_button" token
```

### Query: "Explain what 'cat1a' means"
```
1. Look up "cat1a" in capabilities manifest
2. Answer: "This chip is in the CAT1A family (PSOC 6, FX3G2)."
```

### Query: "What networking options are available?"
```
1. Get all capabilities in "Networking" category
2. List: wifi, ble, bt, ethernet, mesh, audio, ota, ...
3. Show descriptions for each
```

## Data Statistics

From the current manifest:
- **Total capabilities**: ~300+ unique tokens
- **Categories**: ~12 major categories
- **Memory options**: 
  - Flash: ~30 sizes from 8K to 8384K
  - SRAM: ~30 sizes from 2K to 2048K
- **Chip families**: ~40 different families
- **Hardware blocks**: ~40 peripheral types
- **Types**: chip, board, generation

## Common Capability Combinations

### WiFi + BLE Boards (IoT)
```
wifi, ble, psoc6, cat1a, flash_2048k, sram_1024k
```

### Low-Power Sensing Board (PSoC 4)
```
psoc4, cat2, capsense, capsense_button, flash_256k, sram_32k, low_power
```

### USB-C Power Delivery (PMG1)
```
pmg1, cat2, usbpd, flash_128k, sram_16k
```

### Industrial Controller (XMC7000)
```
xmc7000, cat1c, can, ethernet, flash_4192k, sram_1024k, multi_core
```

## Memory Size Patterns

Infineon chips follow power-of-2 sizing, but with some variations:

**Common Flash Sizes:**
- Entry: 32K, 64K, 128K
- Mid-range: 256K, 512K
- High-end: 1024K (1MB), 2048K (2MB), 4096K (4MB)
- Special: 1088K, 1856K, 2112K (includes bootloader space)

**Common SRAM Sizes:**
- Entry: 16K, 32K, 64K
- Mid-range: 128K, 256K
- High-end: 512K, 1024K (1MB), 2048K (2MB)
- Special: 288K, 352K (optimized for specific applications)

## Notes

- Tokens are case-sensitive (usually lowercase)
- Some tokens are families (e.g., `cyw43xxx` covers multiple chips)
- A BSP can have multiple capability tokens
- Not all capabilities apply to all boards
- The manifest grows as new chips/boards are added
- Memory capabilities only indicate on-chip memory, not external
- Some capabilities are mutually exclusive (e.g., you can't be both cat1 and cat2)
