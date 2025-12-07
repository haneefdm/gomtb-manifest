package mtbmanifest

import (
	"fmt"
	"strings"
)

// String returns a formatted version string
// Examples and tests
func SemanticVersionTest() {
	testCases := []string{
		"release-v3.4.0",
		"v2.5.1",
		"3.0.0",
		"latest-v10.X",
		"bmi160_v3.9.1",
		"release-v1.5.0-beta",
		"10.6.201",
		"v5.8.0 (stable)",
		"2.0.0 release",
		"abc-1.2.3-xyz",
		"just-v2.5-test",
		"bmm150_v2.0.0",
	}

	fmt.Println("Version Parsing Examples:")
	fmt.Println(strings.Repeat("=", 60))

	for _, tc := range testCases {
		v, err := ParseVersion(tc)
		if err != nil {
			fmt.Printf("❌ %-25s → Error: %v\n", tc, err)
			continue
		}

		fmt.Printf("✓ %-25s → Major:%d Minor:%d Patch:%d\n",
			tc, v.Major, v.Minor, v.Patch)
		if v.Prefix != "" {
			fmt.Printf("  %25s    Prefix: %q\n", "", v.Prefix)
		}
		if v.Suffix != "" {
			fmt.Printf("  %25s    Suffix: %q\n", "", v.Suffix)
		}
	}

	fmt.Println("\nVersion Comparison Examples:")
	fmt.Println(strings.Repeat("=", 60))

	v1, _ := ParseVersion("release-v3.4.0")
	v2, _ := ParseVersion("release-v3.5.0")
	v3, _ := ParseVersion("v4.0.0")

	fmt.Printf("%s vs %s: %d\n", v1.String(), v2.String(), v1.Compare(v2))
	fmt.Printf("%s vs %s: %d\n", v2.String(), v1.String(), v2.Compare(v1))
	fmt.Printf("%s vs %s: %d\n", v2.String(), v3.String(), v2.Compare(v3))
}
