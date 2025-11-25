// main.go
package main

import (
	"fmt"

	. "github.com/hdm/gomtb-manifest/manifest"
)

var CY_TOOLS_PATH = "/Applications/MoodusToolbox/tools_3.6"
var SuperManifestURL = "https://github.com/Infineon/mtb-super-manifest/raw/v2.X/mtb-super-manifest-fv2.xml"
var ProxyUrl = "" // e.g., "http://user:password@your_proxy_host:your_proxy_port"

func main() {
	boards, err := ingestManifest()
	if err != nil {
		fmt.Printf("Error ingesting manifest: %v\n", err)
		return
	}
	fmt.Printf("Ingested %d boards from manifest.\n", len(boards.Boards))
}

func ingestManifest() (*Boards, error) {
	// Example usage of fetching and reading the super manifest
	fmt.Println("Fetching super manifest...")
	content, err := GetUrlContent(SuperManifestURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch super manifest: %v", err)
	}
	fmt.Printf("Super manifest content length: %d bytes\n", len(content))
	boards, err := ReadSuperManifest(content)
	return boards, nil
}
