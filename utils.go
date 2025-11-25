package gomtb

import (
	"fmt"
	"strings"
)

// ValidateManifest performs basic validation on a manifest
func ValidateManifest(m *Manifest) []error {
	var errors []error

	if m.Name == "" {
		errors = append(errors, fmt.Errorf("manifest name is required"))
	}

	if m.Version == "" {
		errors = append(errors, fmt.Errorf("manifest version is required"))
	}

	// Validate dependencies
	depNames := make(map[string]bool)
	for i, dep := range m.Dependencies {
		if dep.Name == "" {
			errors = append(errors, fmt.Errorf("dependency at index %d has empty name", i))
		} else if depNames[dep.Name] {
			errors = append(errors, fmt.Errorf("duplicate dependency name: %s", dep.Name))
		}
		depNames[dep.Name] = true

		if dep.Version == "" {
			errors = append(errors, fmt.Errorf("dependency %s has empty version", dep.Name))
		}
	}

	// Validate boards
	boardNames := make(map[string]bool)
	for i, board := range m.Boards {
		if board.Name == "" {
			errors = append(errors, fmt.Errorf("board at index %d has empty name", i))
		} else if boardNames[board.Name] {
			errors = append(errors, fmt.Errorf("duplicate board name: %s", board.Name))
		}
		boardNames[board.Name] = true
	}

	// Validate apps
	appNames := make(map[string]bool)
	for i, app := range m.Apps {
		if app.Name == "" {
			errors = append(errors, fmt.Errorf("app at index %d has empty name", i))
		} else if appNames[app.Name] {
			errors = append(errors, fmt.Errorf("duplicate app name: %s", app.Name))
		}
		appNames[app.Name] = true
	}

	return errors
}

// FilterDependencies returns dependencies matching the given filter function
func (m *Manifest) FilterDependencies(filter func(Dependency) bool) []Dependency {
	var result []Dependency
	for _, dep := range m.Dependencies {
		if filter(dep) {
			result = append(result, dep)
		}
	}
	return result
}

// GetRequiredDependencies returns all required dependencies
func (m *Manifest) GetRequiredDependencies() []Dependency {
	return m.FilterDependencies(func(dep Dependency) bool {
		return dep.Required
	})
}

// GetOptionalDependencies returns all optional dependencies
func (m *Manifest) GetOptionalDependencies() []Dependency {
	return m.FilterDependencies(func(dep Dependency) bool {
		return !dep.Required
	})
}

// FilterBoards returns boards matching the given filter function
func (m *Manifest) FilterBoards(filter func(Board) bool) []Board {
	var result []Board
	for _, board := range m.Boards {
		if filter(board) {
			result = append(result, board)
		}
	}
	return result
}

// GetBoardsByChip returns all boards that support a specific chip
func (m *Manifest) GetBoardsByChip(chipName string) []Board {
	return m.FilterBoards(func(board Board) bool {
		for _, chip := range board.Chips {
			if chip == chipName {
				return true
			}
		}
		return false
	})
}

// FilterApps returns apps matching the given filter function
func (m *Manifest) FilterApps(filter func(App) bool) []App {
	var result []App
	for _, app := range m.Apps {
		if filter(app) {
			result = append(result, app)
		}
	}
	return result
}

// Summary returns a human-readable summary of the manifest
func (m *Manifest) Summary() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Manifest: %s (v%s)\n", m.Name, m.Version))
	if m.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", m.Description))
	}
	sb.WriteString(fmt.Sprintf("Dependencies: %d\n", len(m.Dependencies)))
	sb.WriteString(fmt.Sprintf("Boards: %d\n", len(m.Boards)))
	sb.WriteString(fmt.Sprintf("Apps: %d\n", len(m.Apps)))
	return sb.String()
}

// HasDependency checks if a dependency with the given name exists
func (m *Manifest) HasDependency(name string) bool {
	return m.GetDependency(name) != nil
}

// HasBoard checks if a board with the given name exists
func (m *Manifest) HasBoard(name string) bool {
	return m.GetBoard(name) != nil
}

// HasApp checks if an app with the given name exists
func (m *Manifest) HasApp(name string) bool {
	return m.GetApp(name) != nil
}

// RemoveDependency removes a dependency by name, returns true if removed
func (m *Manifest) RemoveDependency(name string) bool {
	for i, dep := range m.Dependencies {
		if dep.Name == name {
			m.Dependencies = append(m.Dependencies[:i], m.Dependencies[i+1:]...)
			return true
		}
	}
	return false
}

// RemoveBoard removes a board by name, returns true if removed
func (m *Manifest) RemoveBoard(name string) bool {
	for i, board := range m.Boards {
		if board.Name == name {
			m.Boards = append(m.Boards[:i], m.Boards[i+1:]...)
			return true
		}
	}
	return false
}

// RemoveApp removes an app by name, returns true if removed
func (m *Manifest) RemoveApp(name string) bool {
	for i, app := range m.Apps {
		if app.Name == name {
			m.Apps = append(m.Apps[:i], m.Apps[i+1:]...)
			return true
		}
	}
	return false
}
