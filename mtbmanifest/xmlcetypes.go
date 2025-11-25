package mtbmanifest

import (
	"encoding/xml"
	"strings"
)

// Code Example Manifest structures
// Handles both mtb-ce-manifest.xml (v1) and mtb-ce-manifest-fv2.xml (v2)

type Apps struct {
	XMLName xml.Name `xml:"apps"`
	Version string   `xml:"version,attr,omitempty"` // Only in v2 (fv2): "2.0"
	App     []App    `xml:"app"`
}

type App struct {
	XMLName           xml.Name   `xml:"app"`
	Keywords          string     `xml:"keywords,attr,omitempty"`            // v2 only: comma-delimited
	ReqCapabilities   string     `xml:"req_capabilities,attr,omitempty"`    // v1: space-delimited string
	ReqCapabilitiesV2 string     `xml:"req_capabilities_v2,attr,omitempty"` // v2: bracketed syntax
	Name              string     `xml:"n"`
	ID                string     `xml:"id"`
	Category          string     `xml:"category,omitempty"` // v2 only
	URI               string     `xml:"uri"`
	Description       string     `xml:"description"`
	Versions          CEVersions `xml:"versions"`
}

type CEVersions struct {
	XMLName xml.Name    `xml:"versions"`
	Version []CEVersion `xml:"version"`
}

type CEVersion struct {
	XMLName                     xml.Name `xml:"version"`
	FlowVersion                 string   `xml:"flow_version,attr,omitempty"`
	ToolsMinVersion             string   `xml:"tools_min_version,attr,omitempty"`               // v2
	ToolsMaxVersion             string   `xml:"tools_max_version,attr,omitempty"`               // v1
	ReqCapabilitiesPerVersion   string   `xml:"req_capabilities_per_version,attr,omitempty"`    // v1: space-delimited
	ReqCapabilitiesPerVersionV2 string   `xml:"req_capabilities_per_version_v2,attr,omitempty"` // v2: bracketed syntax
	Num                         string   `xml:"num"`
	Commit                      string   `xml:"commit"`
}

// CapabilityRequirement represents parsed capability requirements
// For v2 format: groups with OR logic within brackets, AND logic between groups
// For v1 format: simple list of required capabilities (all AND'd together)
type CapabilityRequirement struct {
	// Groups contains capability groups where:
	// - Items within a group are OR'd together (any one matches)
	// - Groups are AND'd together (all groups must match)
	Groups [][]string

	// IsV2 indicates if this was parsed from v2 bracketed syntax
	IsV2 bool
}

// ParseCapabilities parses capability strings from either v1 or v2 format
// v1 format: "psoc6 led capsense_button" (space-delimited, all required)
// v2 format: "[psoc6,t2gbe] hal led [flash_2048k,flash_1024k]" (bracketed OR groups)
func ParseCapabilities(capString string) CapabilityRequirement {
	capString = strings.TrimSpace(capString)
	if capString == "" {
		return CapabilityRequirement{Groups: [][]string{}}
	}

	// Detect v2 format by presence of brackets
	if strings.Contains(capString, "[") {
		return parseV2Capabilities(capString)
	}
	return parseV1Capabilities(capString)
}

// parseV1Capabilities parses space-delimited capability strings
// Each capability is required (implicit AND)
func parseV1Capabilities(capString string) CapabilityRequirement {
	fields := strings.Fields(capString)
	groups := make([][]string, 0, len(fields))

	// Each capability becomes a single-item group (required)
	for _, field := range fields {
		groups = append(groups, []string{field})
	}

	return CapabilityRequirement{
		Groups: groups,
		IsV2:   false,
	}
}

// parseV2Capabilities parses bracketed capability syntax
// Format: "[psoc6,t2gbe] hal led [flash_2048k,flash_1024k]"
// - [a,b,c] = OR group (any one of a, b, or c)
// - plain items = required single capability
// - groups/items are AND'd together
func parseV2Capabilities(capString string) CapabilityRequirement {
	groups := make([][]string, 0)

	// State machine for parsing
	inBracket := false
	current := strings.Builder{}

	for i := 0; i < len(capString); i++ {
		ch := capString[i]

		switch ch {
		case '[':
			// Flush any pending plain text
			if current.Len() > 0 {
				addPlainCapabilities(&groups, current.String())
				current.Reset()
			}
			inBracket = true

		case ']':
			if inBracket {
				// Add bracket group as OR'd capabilities
				orGroup := strings.Split(current.String(), ",")
				cleaned := make([]string, 0, len(orGroup))
				for _, cap := range orGroup {
					if trimmed := strings.TrimSpace(cap); trimmed != "" {
						cleaned = append(cleaned, trimmed)
					}
				}
				if len(cleaned) > 0 {
					groups = append(groups, cleaned)
				}
				current.Reset()
			}
			inBracket = false

		case ' ', '\t', '\n', '\r':
			if !inBracket {
				// Space outside brackets: flush current plain capability
				if current.Len() > 0 {
					addPlainCapabilities(&groups, current.String())
					current.Reset()
				}
			} else {
				// Space inside brackets is ignored (capabilities are comma-separated)
			}

		default:
			current.WriteByte(ch)
		}
	}

	// Flush any remaining plain text
	if current.Len() > 0 {
		addPlainCapabilities(&groups, current.String())
	}

	return CapabilityRequirement{
		Groups: groups,
		IsV2:   true,
	}
}

// addPlainCapabilities adds plain (non-bracketed) capabilities as single-item groups
func addPlainCapabilities(groups *[][]string, text string) {
	fields := strings.Fields(text)
	for _, field := range fields {
		*groups = append(*groups, []string{field})
	}
}

// GetCapabilities returns the parsed capability requirements for an App
// Prefers v2 format if available, falls back to v1
func (a *App) GetCapabilities() CapabilityRequirement {
	if a.ReqCapabilitiesV2 != "" {
		return ParseCapabilities(a.ReqCapabilitiesV2)
	}
	return ParseCapabilities(a.ReqCapabilities)
}

// GetCapabilities returns the parsed capability requirements for a specific version
// Prefers v2 format if available, falls back to v1
func (v *CEVersion) GetCapabilities() CapabilityRequirement {
	if v.ReqCapabilitiesPerVersionV2 != "" {
		return ParseCapabilities(v.ReqCapabilitiesPerVersionV2)
	}
	return ParseCapabilities(v.ReqCapabilitiesPerVersion)
}

// Matches checks if a set of available capabilities satisfies this requirement
// availableCaps should be a set-like structure (use a map for O(1) lookup)
func (cr *CapabilityRequirement) Matches(availableCaps map[string]bool) bool {
	// All groups must be satisfied (AND logic between groups)
	for _, group := range cr.Groups {
		// At least one capability in the group must be available (OR logic within group)
		groupMatched := false
		for _, cap := range group {
			if availableCaps[cap] {
				groupMatched = true
				break
			}
		}
		if !groupMatched {
			return false // This group not satisfied
		}
	}
	return true // All groups satisfied
}

// String returns a human-readable representation of the capability requirement
func (cr *CapabilityRequirement) String() string {
	if len(cr.Groups) == 0 {
		return "(no requirements)"
	}

	parts := make([]string, 0, len(cr.Groups))
	for _, group := range cr.Groups {
		if len(group) == 1 {
			parts = append(parts, group[0])
		} else {
			parts = append(parts, "("+strings.Join(group, " OR ")+")")
		}
	}
	return strings.Join(parts, " AND ")
}

// GetKeywords returns the keywords as a slice, parsed from comma-delimited string
func (a *App) GetKeywords() []string {
	if a.Keywords == "" {
		return []string{}
	}

	keywords := strings.Split(a.Keywords, ",")
	result := make([]string, 0, len(keywords))
	for _, kw := range keywords {
		if trimmed := strings.TrimSpace(kw); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// GetToolsVersion returns the appropriate tools version string (min for v2, max for v1)
func (v *CEVersion) GetToolsVersion() (version string, isMin bool) {
	if v.ToolsMinVersion != "" {
		return v.ToolsMinVersion, true
	}
	return v.ToolsMaxVersion, false
}

// IsV2 checks if this is a v2 format manifest
func (apps *Apps) IsV2() bool {
	return apps.Version == "2.0"
}
