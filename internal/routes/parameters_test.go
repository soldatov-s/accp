package routes

import (
	"net/http"
	"testing"
	"time"

	"github.com/soldatov-s/accp/internal/cache"
	"github.com/soldatov-s/accp/internal/httpclient"
	"github.com/soldatov-s/accp/internal/limits"
	"github.com/soldatov-s/accp/internal/routes/refresh"
	"github.com/soldatov-s/accp/x/helper"
	"github.com/stretchr/testify/require"
)

func TestSetDefault(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "test empty parameters",
			testFunc: func() {
				p := &Parameters{}

				p.SetDefault()
				require.Equal(t, defaultMethods(), p.Methods)
				require.NotNil(t, p.Cache)
				require.True(t, p.Cache.Disabled)
				require.NotNil(t, p.Pool)
				require.NotEmpty(t, p.Pool.Size)
				require.NotEmpty(t, p.Pool.Timeout)
				require.NotNil(t, p.Limits)
				require.Equal(t, 0, len(p.Limits))
			},
		},
		{
			name: "test cache enabled",
			testFunc: func() {
				p := &Parameters{
					Cache: &cache.Config{
						Disabled: false,
					},
				}

				p.SetDefault()
				require.Equal(t, defaultMethods(), p.Methods)
				require.NotNil(t, p.Cache)
				require.NotNil(t, p.Cache.Memory)
				require.NotNil(t, p.Refresh)
				require.NotEmpty(t, p.Refresh.MaxCount)
				require.NotEmpty(t, p.Refresh.Time)
				require.NotNil(t, p.Pool)
				require.NotEmpty(t, p.Pool.Size)
				require.NotEmpty(t, p.Pool.Timeout)
				require.NotNil(t, p.Limits)
				require.Equal(t, 0, len(p.Limits))
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc()
		})
	}
}

// nolint : funlen
func TestMerge(t *testing.T) {
	tests := []struct {
		name               string
		srcParameters      *Parameters
		targetParameters   *Parameters
		expectedParameters *Parameters
	}{
		{
			name:          "src is nil",
			srcParameters: nil,
			targetParameters: func() *Parameters {
				p := &Parameters{
					DSN:                 "test dsn",
					TTL:                 1 * time.Second,
					Limits:              limits.NewMapConfig(),
					Methods:             defaultMethods(),
					Cache:               &cache.Config{},
					Refresh:             &refresh.Config{},
					Pool:                &httpclient.Config{},
					NotIntrospect:       false,
					IntrospectHydration: "nothing",
					RouteKey:            "test key",
				}
				return p
			}(),
			expectedParameters: func() *Parameters {
				p := &Parameters{
					DSN:                 "test dsn",
					TTL:                 1 * time.Second,
					Limits:              limits.NewMapConfig(),
					Methods:             defaultMethods(),
					Cache:               &cache.Config{},
					Refresh:             &refresh.Config{},
					Pool:                &httpclient.Config{},
					NotIntrospect:       false,
					IntrospectHydration: "nothing",
					RouteKey:            "test key",
				}
				return p
			}(),
		},
		{
			name: "target is nil",
			srcParameters: func() *Parameters {
				p := &Parameters{
					DSN:                 "test dsn",
					TTL:                 1 * time.Second,
					Limits:              limits.NewMapConfig(),
					Methods:             defaultMethods(),
					Cache:               &cache.Config{},
					Refresh:             &refresh.Config{},
					Pool:                &httpclient.Config{},
					NotIntrospect:       false,
					IntrospectHydration: "nothing",
					RouteKey:            "test key",
				}
				return p
			}(),
			targetParameters: nil,
			expectedParameters: func() *Parameters {
				p := &Parameters{
					DSN:                 "test dsn",
					TTL:                 1 * time.Second,
					Limits:              limits.NewMapConfig(),
					Methods:             defaultMethods(),
					Cache:               &cache.Config{},
					Refresh:             &refresh.Config{},
					Pool:                &httpclient.Config{},
					NotIntrospect:       false,
					IntrospectHydration: "nothing",
					RouteKey:            "test key",
				}
				return p
			}(),
		},
		{
			name: "target is not nil",
			srcParameters: func() *Parameters {
				p := &Parameters{
					DSN:                 "test dsn",
					TTL:                 1 * time.Second,
					Limits:              limits.NewMapConfig(),
					Methods:             defaultMethods(),
					Cache:               &cache.Config{},
					Refresh:             &refresh.Config{},
					Pool:                &httpclient.Config{},
					NotIntrospect:       false,
					IntrospectHydration: "nothing",
					RouteKey:            "test key",
				}
				return p
			}(),
			targetParameters: func() *Parameters {
				p := &Parameters{
					DSN:                 "test dsn2",
					TTL:                 5 * time.Second,
					Limits:              limits.NewMapConfig(),
					Methods:             helper.Arguments{http.MethodPatch},
					Cache:               &cache.Config{},
					Refresh:             &refresh.Config{},
					Pool:                &httpclient.Config{},
					NotIntrospect:       true,
					IntrospectHydration: "plaintext",
					RouteKey:            "test key2",
				}
				p.Cache.SetDefault()
				p.Refresh.SetDefault()
				p.Pool.SetDefault()
				p.Limits["test"] = &limits.Config{
					Header: []string{"test1"},
				}
				return p
			}(),
			expectedParameters: func() *Parameters {
				p := &Parameters{
					DSN:                 "test dsn2",
					TTL:                 5 * time.Second,
					Limits:              limits.NewMapConfig(),
					Methods:             defaultMethods(),
					Cache:               &cache.Config{},
					Refresh:             &refresh.Config{},
					Pool:                &httpclient.Config{},
					NotIntrospect:       true,
					IntrospectHydration: "plaintext",
					RouteKey:            "test key2",
				}
				p.Cache.SetDefault()
				p.Refresh.SetDefault()
				p.Pool.SetDefault()
				p.Limits["test"] = &limits.Config{
					Header: []string{"test1"},
				}
				p.Methods = append(p.Methods, helper.Arguments{http.MethodPatch}...)
				return p
			}(),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cc := tt.srcParameters.Merge(tt.targetParameters)
			require.Equal(t, tt.expectedParameters, cc)
		})
	}
}
