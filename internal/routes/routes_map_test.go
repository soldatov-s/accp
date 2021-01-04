package routes

import (
	"testing"

	testctxhelper "github.com/soldatov-s/accp/x/test_helpers/ctx"
	"github.com/stretchr/testify/require"
)

func TestAddRouteByPath(t *testing.T) {
	ctx := testctxhelper.InitTestCtx(t)

	rm := make(MapRoutes)

	_, err := rm.AddRouteByPath(ctx, "/api/v1/users", "/api/v1/users", &Parameters{})
	require.Nil(t, err)

	require.Contains(t, rm, "api")
	require.Contains(t, rm["api"].Routes, "v1")
	require.Contains(t, rm["api"].Routes["v1"].Routes, "users")

	r := rm.FindRouteByPath("/api/v1/users")
	require.NotNil(t, r)

	t.Logf("route: %+v", r)

	_, err = rm.AddRouteByPath(ctx, "/api/v1/testers", "/api/v1/testers", &Parameters{})
	require.Nil(t, err)

	r = rm.FindRouteByPath("/api/v1/testers")
	require.NotNil(t, r)

	t.Logf("route: %+v", r)

	_, err = rm.AddRouteByPath(ctx, "/api/v1/testers", "/api/v1/testers", &Parameters{})
	require.NotNil(t, err)
}
