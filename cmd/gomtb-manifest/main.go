// main.go
package main

import (
	"fmt"

	. "github.com/hdm/gomtb-manifest/manifest"
		"github.com/jessevdk/go-flags"

)

var CY_TOOLS_PATH = "/Applications/MoodusToolbox/tools_3.6"
var SuperManifestURL = "https://github.com/Infineon/mtb-super-manifest/raw/v2.X/mtb-super-manifest-fv2.xml"
var ProxyUrl = "" // e.g., "http://user:password@your_proxy_host:your_proxy_port"

var options struct {
	// We should change this to LogLevel or similar later
	Verbose        bool   `short:"v" long:"verbose" description:"Enable verbose logging"`
	UseServer      bool   `short:"s" long:"server" description:"Use an existing server. If none exists, start a new one"`
	OneShot        bool   `short:"o" long:"one-shot" description:"Process file once and exit without server"`
	Port           int    `long:"port" value-name:"PORT" env:"IFXMARKDOWN_PORT" description:"Port for server mode. 0 means force random port"`
	IsContainer    bool   `short:"c" long:"container" description:"Generate container output"`
	ContainerClass string `short:"C" long:"container-class" value-name:"CLASS" description:"CSS class for container div" default:"embedded-md-container"`
	OutputFile     string `short:"O" long:"output" value-name:"FILE" description:"Output file for HTML (defaults to stdout)"`
	NoBrowser      bool   `short:"n" long:"no-browser" description:"Do not open the browser automatically"`
	showHelp       bool   `short:"h" long:"help" description:"Show help message"`
}

func main() {
	_, err := flags.Parse(&options)
	if err != nil {
		fmt.Printf("Error parsing command-line options: %v\n", err)
		return
	}
	if options.showHelp {
		flags.NewParser(&options, flags.Default).WriteHelp(fmt.Writer(os.Stdout))
		return
	}

	// For demonstration, we will just ingest the manifest and print the number of boards	
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
