package mtbmanifest

import (
	"encoding/xml"
	"testing"
)

func TestParseV1Capabilities(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int // number of groups
		wantV2   bool
	}{
		{
			name:     "simple v1",
			input:    "psoc6 led",
			expected: 2,
			wantV2:   false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
			wantV2:   false,
		},
		{
			name:     "single capability",
			input:    "psoc6",
			expected: 1,
			wantV2:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCapabilities(tt.input)
			if len(result.Groups) != tt.expected {
				t.Errorf("expected %d groups, got %d", tt.expected, len(result.Groups))
			}
			if result.IsV2 != tt.wantV2 {
				t.Errorf("expected IsV2=%v, got %v", tt.wantV2, result.IsV2)
			}
		})
	}
}

func TestParseV2Capabilities(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedStr  string
		groupLengths []int
	}{
		{
			name:         "single OR group",
			input:        "[psoc6,t2gbe]",
			expectedStr:  "(psoc6 OR t2gbe)",
			groupLengths: []int{2},
		},
		{
			name:         "mixed plain and OR",
			input:        "hal [psoc6,t2gbe] led",
			expectedStr:  "hal AND (psoc6 OR t2gbe) AND led",
			groupLengths: []int{1, 2, 1},
		},
		{
			name:         "multiple OR groups",
			input:        "[cat1a,cat1b] [flash_2048k,flash_1024k] hal",
			expectedStr:  "(cat1a OR cat1b) AND (flash_2048k OR flash_1024k) AND hal",
			groupLengths: []int{2, 2, 1},
		},
		{
			name:         "complex example from manifest",
			input:        "hal led [psoc6,t2gbe,xmc7000] [flash_0k,flash_2048k]",
			expectedStr:  "hal AND led AND (psoc6 OR t2gbe OR xmc7000) AND (flash_0k OR flash_2048k)",
			groupLengths: []int{1, 1, 3, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCapabilities(tt.input)

			if !result.IsV2 {
				t.Error("expected IsV2=true")
			}

			if len(result.Groups) != len(tt.groupLengths) {
				t.Errorf("expected %d groups, got %d", len(tt.groupLengths), len(result.Groups))
			}

			for i, expectedLen := range tt.groupLengths {
				if i >= len(result.Groups) {
					break
				}
				if len(result.Groups[i]) != expectedLen {
					t.Errorf("group %d: expected length %d, got %d", i, expectedLen, len(result.Groups[i]))
				}
			}

			if result.String() != tt.expectedStr {
				t.Errorf("expected string %q, got %q", tt.expectedStr, result.String())
			}
		})
	}
}

func TestCapabilityMatching(t *testing.T) {
	tests := []struct {
		name      string
		capString string
		available map[string]bool
		matches   bool
	}{
		{
			name:      "v1 simple match",
			capString: "psoc6 led",
			available: map[string]bool{"psoc6": true, "led": true, "other": true},
			matches:   true,
		},
		{
			name:      "v1 missing capability",
			capString: "psoc6 led",
			available: map[string]bool{"psoc6": true, "other": true},
			matches:   false,
		},
		{
			name:      "v2 OR match first option",
			capString: "[psoc6,t2gbe] led",
			available: map[string]bool{"psoc6": true, "led": true},
			matches:   true,
		},
		{
			name:      "v2 OR match second option",
			capString: "[psoc6,t2gbe] led",
			available: map[string]bool{"t2gbe": true, "led": true},
			matches:   true,
		},
		{
			name:      "v2 OR no match",
			capString: "[psoc6,t2gbe] led",
			available: map[string]bool{"xmc7000": true, "led": true},
			matches:   false,
		},
		{
			name:      "v2 complex match",
			capString: "hal [psoc6,t2gbe] [flash_2048k,flash_1024k]",
			available: map[string]bool{"hal": true, "t2gbe": true, "flash_1024k": true},
			matches:   true,
		},
		{
			name:      "v2 complex missing one group",
			capString: "hal [psoc6,t2gbe] [flash_2048k,flash_1024k]",
			available: map[string]bool{"hal": true, "t2gbe": true, "flash_512k": true},
			matches:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := ParseCapabilities(tt.capString)
			result := caps.Matches(tt.available)
			if result != tt.matches {
				t.Errorf("expected match=%v, got %v for caps %s", tt.matches, result, caps.String())
			}
		})
	}
}

func TestAppStructParsing(t *testing.T) {
	v1XML := `<apps>
  <app>
    <n>Test App</n>
    <id>test-app-1</id>
    <uri>https://example.com</uri>
    <description>Test description</description>
    <req_capabilities>psoc6 led</req_capabilities>
    <versions>
      <version tools_max_version="2.1.0">
        <num>1.0.0</num>
        <commit>v1.0.0</commit>
      </version>
    </versions>
  </app>
</apps>`

	var apps Apps
	if err := xml.Unmarshal([]byte(v1XML), &apps); err != nil {
		t.Fatalf("failed to parse XML: %v", err)
	}

	if len(apps.App) != 1 {
		t.Fatalf("expected 1 app, got %d", len(apps.App))
	}

	app := apps.App[0]
	if app.Name != "Test App" {
		t.Errorf("expected name 'Test App', got %q", app.Name)
	}

	caps := app.GetCapabilities()
	if caps.IsV2 {
		t.Error("expected v1 format")
	}
	if len(caps.Groups) != 2 {
		t.Errorf("expected 2 capability groups, got %d", len(caps.Groups))
	}
}

func TestV2AppParsing(t *testing.T) {
	v2XML := `<apps version="2.0">
  <app keywords="led,starter" req_capabilities_v2="hal [psoc6,t2gbe]">
    <n>Test App V2</n>
    <id>test-app-v2</id>
    <category>Testing</category>
    <uri>https://example.com</uri>
    <description>Test description</description>
    <versions>
      <version flow_version="2.0" tools_min_version="3.0.0" req_capabilities_per_version_v2="[bsp_gen4]">
        <num>2.0.0</num>
        <commit>v2.0.0</commit>
      </version>
    </versions>
  </app>
</apps>`

	var apps Apps
	if err := xml.Unmarshal([]byte(v2XML), &apps); err != nil {
		t.Fatalf("failed to parse XML: %v", err)
	}

	if !apps.IsV2() {
		t.Error("expected v2 manifest")
	}

	app := apps.App[0]
	if app.Category != "Testing" {
		t.Errorf("expected category 'Testing', got %q", app.Category)
	}

	keywords := app.GetKeywords()
	if len(keywords) != 2 {
		t.Errorf("expected 2 keywords, got %d", len(keywords))
	}

	caps := app.GetCapabilities()
	if !caps.IsV2 {
		t.Error("expected v2 capability format")
	}

	version := app.Versions.Version[0]
	toolsVer, isMin := version.GetToolsVersion()
	if !isMin {
		t.Error("expected minimum version")
	}
	if toolsVer != "3.0.0" {
		t.Errorf("expected tools version '3.0.0', got %q", toolsVer)
	}
}

func TestKeywordsParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple keywords",
			input:    "led,starter,hello world",
			expected: []string{"led", "starter", "hello world"},
		},
		{
			name:     "keywords with spaces",
			input:    "led, starter, hello world",
			expected: []string{"led", "starter", "hello world"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single keyword",
			input:    "led",
			expected: []string{"led"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := App{Keywords: tt.input}
			result := app.GetKeywords()

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d keywords, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("keyword %d: expected %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}
