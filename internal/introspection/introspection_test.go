package introspection

import (
	"context"
	"testing"
	"time"

	"github.com/soldatov-s/accp/internal/httpclient"
	"github.com/stretchr/testify/require"
)

func initTestIntrospector(t *testing.T) *Introspect {
	ctx := context.Background()
	ic := &Config{
		DSN:            "http://localhost:8001",
		Endpoint:       "/oauth2/introspect",
		ContentType:    "application/x-www-form-urlencoded",
		Method:         "POST",
		ValidMarker:    `"active":true`,
		BodyTemplate:   `token_type_hint=access_token&token={{.Token}}`,
		CookieName:     []string{"access-token"},
		QueryParamName: []string{"access_token"},
		Pool: &httpclient.Config{
			Size:    50,
			Timeout: 10 * time.Second,
		},
	}

	i, err := NewIntrospector(ctx, ic)
	require.Nil(t, err)

	return i
}

func TestTrimFields(t *testing.T) {
	i := initTestIntrospector(t)

	i.cfg.TrimmedFilds = []string{"exp", "iat"}
	i.initRegex()

	tests := []struct {
		name    string
		content []byte
	}{
		{
			name:    "1-st type of body",
			content: []byte(`{"token":"token","token_type":"bearer","exp":12345,"iat":123456}`),
		},
		{
			name:    "2-nd type of body",
			content: []byte(`{"exp":12345,"iat":123456,"token":"token","token_type":"bearer"}`),
		},
		{
			name:    "3-ed type of body",
			content: []byte(`{"token":"token","exp":12345,"iat":123456,"token_type":"bearer"}`),
		},
		{
			name:    "4-th type of body",
			content: []byte(`{"token":"token","exp":12345,"token_type":"bearer","iat":123456}`),
		},
	}

	expectedContent := []byte(`{"token":"token","token_type":"bearer"}`)
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("introspect response before trim: %s", string(tt.content))
			content := i.trimFields(tt.content)
			t.Logf("introspect response after trim: %s", string(content))
			require.Equal(t, expectedContent, content)
		})
	}
}
