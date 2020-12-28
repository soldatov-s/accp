package routes

import (
	"fmt"
	"net/http"
	"strings"
)

type MapRoutes map[string]*Route

func (m MapRoutes) findRoute(path string) *Route {
	path = strings.Trim(path, "/")
	strs := strings.Split(path, "/")
	var (
		route *Route
		ok    bool
	)

	tmp := m

	for _, s := range strs {
		if s == "" {
			continue
		}
		if route, ok = tmp[s]; !ok {
			return route
		}
		tmp = route.Routes
	}

	return route
}

func (m MapRoutes) FindRouteByPath(path string) *Route {
	return m.findRoute(path)
}

func (m MapRoutes) FindRouteByHTTPRequest(r *http.Request) *Route {
	return m.FindRouteByPath(r.URL.Path)
}

func (m MapRoutes) FindDuplicated(path string) error {
	if m.FindRouteByPath(path) != nil {
		return fmt.Errorf("duplicated excluded route: %s", path)
	}

	return nil
}
