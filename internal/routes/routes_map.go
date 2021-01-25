package routes

import (
	"context"
	"net/http"
	"strings"
)

type MapRoutes map[string]*Route

func (m MapRoutes) FindRouteByPath(path string) *Route {
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

func (m MapRoutes) FindRouteByHTTPRequest(r *http.Request) *Route {
	return m.FindRouteByPath(r.URL.Path)
}

func (m MapRoutes) AddRouteByPath(ctx context.Context, path, routeName string, params *Parameters) (*Route, error) {
	path = strings.Trim(path, "/")
	strs := strings.Split(path, "/")
	var (
		route *Route
		ok    bool
	)

	var previousLevelRoutes MapRoutes
	tmp := m

	for i, s := range strs {
		if s == "" {
			continue
		}
		if route, ok = tmp[s]; !ok {
			tmp[s] = NewRoute(ctx, routeName, params)
			previousLevelRoutes = tmp
			tmp = tmp[s].Routes
		} else {
			if i+1 == len(strs) {
				return nil, &ErrDuplicatedRoute{route: path}
			}
			tmp = route.Routes
		}
	}

	lastPartOfRoute := strs[len(strs)-1]
	if lastPartOfRoute == "" {
		lastPartOfRoute = strs[len(strs)-2]
	}

	return previousLevelRoutes[lastPartOfRoute], nil
}
