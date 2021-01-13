package introspection

import (
	"github.com/soldatov-s/accp/internal/errors"
	"github.com/soldatov-s/accp/internal/httpclient"
)

type Config struct {
	// DSN is DSN your authorization service for introspection
	DSN string
	// Endpoint is intropsection endpoint on introspection service
	Endpoint string
	// ContentType - content type of request
	ContentType string
	// Method - REST API method
	Method string
	// ValidMarker - marker in answer that shows that token is valid
	ValidMarker string
	// BodyTmpl - teplate of body
	BodyTemplate string
	// TrimmedFilds - trimed fields in answer from introspector
	TrimmedFilds []string
	// CookieName is a list of cookie where may be stored access token
	CookieName []string
	// QueryParamName is a list of query parameter where may be stored access token
	QueryParamName []string
	// Pool is config for http clients pool
	Pool *httpclient.Config
}

func (c *Config) Validate() error {
	if c.DSN == "" {
		return errors.EmptyConfigParameter("dsn")
	}

	if c.Endpoint == "" {
		return errors.EmptyConfigParameter("endpoint")
	}

	if c.ContentType == "" {
		return errors.EmptyConfigParameter("contenttype")
	}

	if c.Method == "" {
		return errors.EmptyConfigParameter("method")
	}

	if c.ValidMarker == "" {
		return errors.EmptyConfigParameter("validmarker")
	}

	if c.BodyTemplate == "" {
		return errors.EmptyConfigParameter("bodytemplate")
	}

	return nil
}
