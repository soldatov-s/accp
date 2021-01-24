package routes

import "sort"

// Config declares a route configuration
type Config struct {
	// Parameters are parameters of route
	Parameters *Parameters
	// Routes are subroutes of route
	Routes MapConfig
	// Excluded is an excluded subroutes from route
	Excluded []string
}

type MapConfig map[string]*Config

// SortKeys returns the slice with sorted keys
func (m MapConfig) SortKeys() []string {
	// Sort config map
	keys := make([]string, 0, len(m))

	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}
