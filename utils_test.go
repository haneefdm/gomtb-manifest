package gomtb

import (
	"strings"
	"testing"
)

func TestValidateManifest(t *testing.T) {
	tests := []struct {
		name           string
		manifest       *Manifest
		expectedErrors int
	}{
		{
			name: "Valid manifest",
			manifest: &Manifest{
				Name:    "test",
				Version: "1.0.0",
				Dependencies: []Dependency{
					{Name: "dep1", Version: "1.0.0"},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "Missing name",
			manifest: &Manifest{
				Version: "1.0.0",
			},
			expectedErrors: 1,
		},
		{
			name: "Missing version",
			manifest: &Manifest{
				Name: "test",
			},
			expectedErrors: 1,
		},
		{
			name: "Duplicate dependency names",
			manifest: &Manifest{
				Name:    "test",
				Version: "1.0.0",
				Dependencies: []Dependency{
					{Name: "dep1", Version: "1.0.0"},
					{Name: "dep1", Version: "2.0.0"},
				},
			},
			expectedErrors: 1,
		},
		{
			name: "Empty dependency name",
			manifest: &Manifest{
				Name:    "test",
				Version: "1.0.0",
				Dependencies: []Dependency{
					{Name: "", Version: "1.0.0"},
				},
			},
			expectedErrors: 1,
		},
		{
			name: "Empty dependency version",
			manifest: &Manifest{
				Name:    "test",
				Version: "1.0.0",
				Dependencies: []Dependency{
					{Name: "dep1", Version: ""},
				},
			},
			expectedErrors: 1,
		},
		{
			name: "Duplicate board names",
			manifest: &Manifest{
				Name:    "test",
				Version: "1.0.0",
				Boards: []Board{
					{Name: "board1"},
					{Name: "board1"},
				},
			},
			expectedErrors: 1,
		},
		{
			name: "Duplicate app names",
			manifest: &Manifest{
				Name:    "test",
				Version: "1.0.0",
				Apps: []App{
					{Name: "app1"},
					{Name: "app1"},
				},
			},
			expectedErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateManifest(tt.manifest)
			if len(errors) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d: %v", tt.expectedErrors, len(errors), errors)
			}
		})
	}
}

func TestGetRequiredDependencies(t *testing.T) {
	manifest := &Manifest{
		Dependencies: []Dependency{
			{Name: "req1", Required: true},
			{Name: "opt1", Required: false},
			{Name: "req2", Required: true},
		},
	}

	required := manifest.GetRequiredDependencies()
	if len(required) != 2 {
		t.Errorf("Expected 2 required dependencies, got %d", len(required))
	}
}

func TestGetOptionalDependencies(t *testing.T) {
	manifest := &Manifest{
		Dependencies: []Dependency{
			{Name: "req1", Required: true},
			{Name: "opt1", Required: false},
			{Name: "opt2", Required: false},
		},
	}

	optional := manifest.GetOptionalDependencies()
	if len(optional) != 2 {
		t.Errorf("Expected 2 optional dependencies, got %d", len(optional))
	}
}

func TestGetBoardsByChip(t *testing.T) {
	manifest := &Manifest{
		Boards: []Board{
			{Name: "board1", Chips: []string{"PSoC6", "WiFi"}},
			{Name: "board2", Chips: []string{"PSoC4"}},
			{Name: "board3", Chips: []string{"PSoC6", "BLE"}},
		},
	}

	boards := manifest.GetBoardsByChip("PSoC6")
	if len(boards) != 2 {
		t.Errorf("Expected 2 boards with PSoC6, got %d", len(boards))
	}

	boards = manifest.GetBoardsByChip("PSoC4")
	if len(boards) != 1 {
		t.Errorf("Expected 1 board with PSoC4, got %d", len(boards))
	}

	boards = manifest.GetBoardsByChip("NonExistent")
	if len(boards) != 0 {
		t.Errorf("Expected 0 boards with NonExistent chip, got %d", len(boards))
	}
}

func TestSummary(t *testing.T) {
	manifest := &Manifest{
		Name:         "test-manifest",
		Version:      "1.0.0",
		Description:  "Test Description",
		Dependencies: []Dependency{{Name: "dep1"}},
		Boards:       []Board{{Name: "board1"}},
		Apps:         []App{{Name: "app1"}},
	}

	summary := manifest.Summary()
	if !strings.Contains(summary, "test-manifest") {
		t.Error("Summary should contain manifest name")
	}
	if !strings.Contains(summary, "1.0.0") {
		t.Error("Summary should contain version")
	}
	if !strings.Contains(summary, "Test Description") {
		t.Error("Summary should contain description")
	}
	if !strings.Contains(summary, "Dependencies: 1") {
		t.Error("Summary should contain dependency count")
	}
}

func TestHasDependency(t *testing.T) {
	manifest := &Manifest{
		Dependencies: []Dependency{
			{Name: "dep1"},
		},
	}

	if !manifest.HasDependency("dep1") {
		t.Error("Should find existing dependency")
	}
	if manifest.HasDependency("nonexistent") {
		t.Error("Should not find nonexistent dependency")
	}
}

func TestHasBoard(t *testing.T) {
	manifest := &Manifest{
		Boards: []Board{
			{Name: "board1"},
		},
	}

	if !manifest.HasBoard("board1") {
		t.Error("Should find existing board")
	}
	if manifest.HasBoard("nonexistent") {
		t.Error("Should not find nonexistent board")
	}
}

func TestHasApp(t *testing.T) {
	manifest := &Manifest{
		Apps: []App{
			{Name: "app1"},
		},
	}

	if !manifest.HasApp("app1") {
		t.Error("Should find existing app")
	}
	if manifest.HasApp("nonexistent") {
		t.Error("Should not find nonexistent app")
	}
}

func TestRemoveDependency(t *testing.T) {
	manifest := &Manifest{
		Dependencies: []Dependency{
			{Name: "dep1"},
			{Name: "dep2"},
		},
	}

	if !manifest.RemoveDependency("dep1") {
		t.Error("Should successfully remove existing dependency")
	}
	if len(manifest.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency after removal, got %d", len(manifest.Dependencies))
	}
	if manifest.Dependencies[0].Name != "dep2" {
		t.Error("Wrong dependency was removed")
	}

	if manifest.RemoveDependency("nonexistent") {
		t.Error("Should return false for nonexistent dependency")
	}
}

func TestRemoveBoard(t *testing.T) {
	manifest := &Manifest{
		Boards: []Board{
			{Name: "board1"},
			{Name: "board2"},
		},
	}

	if !manifest.RemoveBoard("board1") {
		t.Error("Should successfully remove existing board")
	}
	if len(manifest.Boards) != 1 {
		t.Errorf("Expected 1 board after removal, got %d", len(manifest.Boards))
	}
	if manifest.Boards[0].Name != "board2" {
		t.Error("Wrong board was removed")
	}

	if manifest.RemoveBoard("nonexistent") {
		t.Error("Should return false for nonexistent board")
	}
}

func TestRemoveApp(t *testing.T) {
	manifest := &Manifest{
		Apps: []App{
			{Name: "app1"},
			{Name: "app2"},
		},
	}

	if !manifest.RemoveApp("app1") {
		t.Error("Should successfully remove existing app")
	}
	if len(manifest.Apps) != 1 {
		t.Errorf("Expected 1 app after removal, got %d", len(manifest.Apps))
	}
	if manifest.Apps[0].Name != "app2" {
		t.Error("Wrong app was removed")
	}

	if manifest.RemoveApp("nonexistent") {
		t.Error("Should return false for nonexistent app")
	}
}

func TestFilterDependencies(t *testing.T) {
	manifest := &Manifest{
		Dependencies: []Dependency{
			{Name: "dep1", Version: "1.0.0"},
			{Name: "dep2", Version: "2.0.0"},
			{Name: "dep3", Version: "1.0.0"},
		},
	}

	filtered := manifest.FilterDependencies(func(dep Dependency) bool {
		return dep.Version == "1.0.0"
	})

	if len(filtered) != 2 {
		t.Errorf("Expected 2 filtered dependencies, got %d", len(filtered))
	}
}

func TestFilterBoards(t *testing.T) {
	manifest := &Manifest{
		Boards: []Board{
			{Name: "board1", Version: "1.0.0"},
			{Name: "board2", Version: "2.0.0"},
			{Name: "board3", Version: "1.0.0"},
		},
	}

	filtered := manifest.FilterBoards(func(board Board) bool {
		return board.Version == "1.0.0"
	})

	if len(filtered) != 2 {
		t.Errorf("Expected 2 filtered boards, got %d", len(filtered))
	}
}

func TestFilterApps(t *testing.T) {
	manifest := &Manifest{
		Apps: []App{
			{Name: "app1", Version: "1.0.0"},
			{Name: "app2", Version: "2.0.0"},
			{Name: "app3", Version: "1.0.0"},
		},
	}

	filtered := manifest.FilterApps(func(app App) bool {
		return app.Version == "1.0.0"
	})

	if len(filtered) != 2 {
		t.Errorf("Expected 2 filtered apps, got %d", len(filtered))
	}
}
