package routes

// Config declares a route configuration
type Config struct {
	// Parameters are parameters of route
	Parameters *Parameters
	// Routes are subroutes of route
	Routes map[string]*Config
	// Excluded is an exluded subroutes from route
	Excluded []string
}
