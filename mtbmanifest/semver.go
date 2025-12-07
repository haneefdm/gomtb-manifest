package mtbmanifest

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Version pattern with optional prefix/suffix and optional patch
// Matches: release-v3.4.0, v2.5, 3.0.0-beta, latest-v10.X, bmi160_v3.9.1, etc.
// Major is mandatory, Minor and Patch can be "X" or missing
var versionRegex = regexp.MustCompile(`(\d+)\.(\d+|X)(?:\.(\d+|X))?`)

// SemanticVersion represents a parsed version
type SemanticVersion struct {
	Raw    string // Original string: "release-v3.4.0"
	Prefix string // "release-v"
	Major  int    // 3
	// With the Minor and Patch fields as int, "X" is represented as -1
	Minor  int    // 4 (or -1 if not present or is "X")
	Patch  int    // 0 (or -1 if not present or is "X")
	Suffix string // "" (or any trailing text)
}

// ParseVersion extracts version numbers from any string with arbitrary prefix/suffix
func ParseVersion(version string) (*SemanticVersion, error) {
	// Find the version pattern
	upperVersion := strings.ToUpper(version)
	matches := versionRegex.FindStringSubmatch(upperVersion)
	if matches == nil {
		return nil, fmt.Errorf("no version pattern found in: %s", version)
	}

	for i := 2; i < 4; i++ {
		if i >= len(matches) || matches[i] == "" {
			matches = append(matches, "-1") // Missing minor/patch treated as -1
		} else if matches[i] == "X" {
			matches[i] = "-1" // Represent "X" as -1
		}
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	// Find where the version match occurred to extract prefix/suffix
	loc := versionRegex.FindStringIndex(version)
	prefix := version[:loc[0]]
	suffix := version[loc[1]:]

	return &SemanticVersion{
		Raw:    version,
		Prefix: prefix,
		Major:  major,
		Minor:  minor,
		Patch:  patch,
		Suffix: suffix,
	}, nil
}

// String returns a formatted version string
func (v *SemanticVersion) String() string {
	if (v.Minor == -1) && (v.Patch == -1) {
		return fmt.Sprintf("%s%d.X%s", v.Prefix, v.Major, v.Suffix)
	}
	if v.Patch == -1 {
		return fmt.Sprintf("%s%d.%d.X%s", v.Prefix, v.Major, v.Minor, v.Suffix)
	}
	return fmt.Sprintf("%s%d.%d.%d%s", v.Prefix, v.Major, v.Minor, v.Patch, v.Suffix)
}

// Compare compares two semantic versions
func wildCmp(a, b int) int {
	if (a == b) || (b == -1) /*|| (a == -1)*/ {
		return 0
	}
	return a - b
}

func (v *SemanticVersion) Compare(other *SemanticVersion) int {
	if majCmp := wildCmp(v.Major, other.Major); majCmp != 0 {
		return majCmp
	}
	if minCmp := wildCmp(v.Minor, other.Minor); minCmp != 0 {
		return minCmp
	}
	if patCmp := wildCmp(v.Patch, other.Patch); patCmp != 0 {
		return patCmp
	}
	return 0
}
