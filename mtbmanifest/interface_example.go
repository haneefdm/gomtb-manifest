package mtbmanifest

// Example demonstrating how to use the SuperManifestIF interface
//
// This file shows how clients can work with SuperManifestIF without
// needing to know about the concrete SuperManifest implementation.

// ProcessSuperManifest is an example function that accepts the interface
// and can work with any implementation of SuperManifestIF
func ProcessSuperManifest(sm SuperManifestIF, boardID string) (*Board, error) {
	// Get all boards using the interface method
	boardsMap := sm.GetBoardsMap()

	// Look up the specific board
	board, exists := (*boardsMap)[boardID]
	if !exists {
		return nil, nil
	}

	// Use interface methods to get BSP information
	if board.Origin != nil && board.Origin.DependencyURL != "" {
		board.BSPDependencies, _ = sm.GetBSPDependencies(board.Origin.DependencyURL, board.ID)
	}

	if board.Origin != nil && board.Origin.CapabilityURL != "" {
		board.BSPCapabilities, _ = sm.GetBSPCapabilitiesManifest(board.Origin.CapabilityURL)
	}

	return board, nil
}

// GetBoardCount is a simple example that counts boards through the interface
func GetBoardCount(sm SuperManifestIF) int {
	boardsMap := sm.GetBoardsMap()
	return len(*boardsMap)
}

// GetAppCount is a simple example that counts apps through the interface
func GetAppCount(sm SuperManifestIF) int {
	appsMap := sm.GetAppsMap()
	return len(*appsMap)
}

// GetMiddlewareCount is a simple example that counts middleware through the interface
func GetMiddlewareCount(sm SuperManifestIF) int {
	middlewareMap := sm.GetMiddlewareMap()
	return len(*middlewareMap)
}

// Example usage pattern:
//
//   func main() {
//       // Create a SuperManifest (returns interface)
//       manifest := mtbmanifest.NewSuperManifest()
//
//       // Or ingest from URL (also returns interface)
//       manifest, err := mtbmanifest.IngestManifestTree("")
//       if err != nil {
//           log.Fatal(err)
//       }
//
//       // Work with the interface
//       boardCount := mtbmanifest.GetBoardCount(manifest)
//       fmt.Printf("Total boards: %d\n", boardCount)
//
//       // Process a specific board
//       board, _ := mtbmanifest.ProcessSuperManifest(manifest, "KIT_PSE84_EVAL_EPC2")
//       if board != nil {
//           fmt.Printf("Found board: %s\n", board.Name)
//       }
//   }
