package introspection

import (
	"testing"

	"github.com/soldatov-s/accp/internal/errors"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	var err error
	c := &Config{}
	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "empty dsn",
			testFunc: func() {
				err = c.Validate()
				require.NotNil(t, err)
				require.Equal(t, errors.EmptyConfigParameter("dsn"), err)
			},
		},
		{
			name: "empty endpoint",
			testFunc: func() {
				c.DSN = "dsn"
				err = c.Validate()
				require.NotNil(t, err)
				require.Equal(t, errors.EmptyConfigParameter("endpoint"), err)
			},
		},
		{
			name: "empty contenttype",
			testFunc: func() {
				c.Endpoint = "endpoint"
				err = c.Validate()
				require.NotNil(t, err)
				require.Equal(t, errors.EmptyConfigParameter("contenttype"), err)
			},
		},
		{
			name: "empty method",
			testFunc: func() {
				c.ContentType = "contenttype"
				err = c.Validate()
				require.NotNil(t, err)
				require.Equal(t, errors.EmptyConfigParameter("method"), err)
			},
		},
		{
			name: "empty validmarker",
			testFunc: func() {
				c.Method = "method"
				err = c.Validate()
				require.NotNil(t, err)
				require.Equal(t, errors.EmptyConfigParameter("validmarker"), err)
			},
		},
		{
			name: "empty bodytemplate",
			testFunc: func() {
				c.ValidMarker = "validmarker"
				err = c.Validate()
				require.NotNil(t, err)
				require.Equal(t, errors.EmptyConfigParameter("bodytemplate"), err)
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
