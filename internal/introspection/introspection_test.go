package introspection

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrimFields(t *testing.T) {
	i := Introspect{}

	i.cfg = &Config{
		TrimmedFilds: []string{"exp", "iat"},
	}
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
