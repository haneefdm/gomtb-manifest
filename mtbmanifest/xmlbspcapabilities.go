package mtbmanifest

import "encoding/json"

// BSPCapabilitiesManifest represents the root capabilities manifest structure
// Example URL: https://raw.githubusercontent.com/Infineon/mtb-bsp-manifest/v2.X/mtb-bsp-capabilities-manifest.json
type BSPCapabilitiesManifest struct {
	// Capabilities is the list of all capability definitions
	Capabilities []*BSPCapability `json:"capabilities"`
}

// BSPCapability represents a single capability definition
type BSPCapability struct {
	// Category groups similar capabilities together
	// Examples: "Chip Families", "Hardware Blocks", "Networking", "Memory", "Human Interface Devices"
	Category string `json:"category"`

	// Description provides a human-readable explanation of this capability
	Description string `json:"description"`

	// Name is a human-readable name for this capability
	// Example: "CYW208XX", "MAX44009EDT", "adc", "arduino"
	Name string `json:"name"`

	// Token is the unique identifier used in BSP manifests to reference this capability
	// This is what appears in the <capabilities> section of BSP manifest XML
	// Example: "CYW208XX", "flash_256k", "sram_128k", "ble"
	Token string `json:"token"`

	// Types indicates where this capability can be applied
	// Common values: "chip", "board", "generation"
	Types []string `json:"types"`
}

// Helper function to find a capability by token
func (m *BSPCapabilitiesManifest) GetCapability(token string) (*BSPCapability, bool) {
	for i := range m.Capabilities {
		if m.Capabilities[i].Token == token {
			return m.Capabilities[i], true
		}
	}
	return nil, false
}

// Helper function to get all capabilities in a category
func (m *BSPCapabilitiesManifest) GetCapabilitiesByCategory(category string) []*BSPCapability {
	var result []*BSPCapability
	for _, cap := range m.Capabilities {
		if cap.Category == category {
			result = append(result, cap)
		}
	}
	return result
}

// Helper function to get all capabilities of a specific type
func (m *BSPCapabilitiesManifest) GetCapabilitiesByType(capType string) []*BSPCapability {
	var result []*BSPCapability
	for _, cap := range m.Capabilities {
		for _, t := range cap.Types {
			if t == capType {
				result = append(result, cap)
				break
			}
		}
	}
	return result
}

// Helper function to get all capability categories
func (m *BSPCapabilitiesManifest) GetCategories() []string {
	categorySet := make(map[string]bool)
	for _, cap := range m.Capabilities {
		categorySet[cap.Category] = true
	}

	categories := make([]string, 0, len(categorySet))
	for cat := range categorySet {
		categories = append(categories, cat)
	}
	return categories
}

// Helper function to find capabilities matching a pattern (case-insensitive substring match)
func (m *BSPCapabilitiesManifest) SearchCapabilities(query string) []*BSPCapability {
	var result []*BSPCapability
	query_lower := toLower(query)

	for _, cap := range m.Capabilities {
		if contains(toLower(cap.Name), query_lower) ||
			contains(toLower(cap.Token), query_lower) ||
			contains(toLower(cap.Description), query_lower) {
			result = append(result, cap)
		}
	}
	return result
}

// Helper function to validate a capability token exists
func (m *BSPCapabilitiesManifest) ValidateToken(token string) bool {
	_, found := m.GetCapability(token)
	return found
}

// Helper function to explain multiple capability tokens
func (m *BSPCapabilitiesManifest) ExplainTokens(tokens []string) map[string]string {
	explanations := make(map[string]string)
	for _, token := range tokens {
		if cap, found := m.GetCapability(token); found {
			explanations[token] = cap.Description
		} else {
			explanations[token] = "Unknown capability"
		}
	}
	return explanations
}

// Simple string helpers (Go 1.x compatible)
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		result[i] = c
	}
	return string(result)
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func ReadBSPCapabilitiesManifest(data []byte) (*BSPCapabilitiesManifest, error) {
	var manifest BSPCapabilitiesManifest
	err := json.Unmarshal(data, &manifest)
	if err != nil {
		return nil, err
	}
	return &manifest, nil
}
