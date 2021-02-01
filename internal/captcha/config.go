package captcha

import (
	"time"

	"github.com/soldatov-s/accp/internal/errors"
	"github.com/soldatov-s/accp/internal/httpclient"
	"github.com/soldatov-s/accp/x/helper"
)

const (
	defaultTokenTTL = 30 * 24 * time.Hour
)

// Config is config structure for captcha
type Config struct {
	// JWT token TTL
	TokenTTL time.Duration
	// JWT token sign
	TokenSign string
	// Google verify URL
	VerifyURL string
	// Google captcha secret key
	Key string
	// Header is name of header in request for captcha
	Header helper.Arguments
	// Cookie is name of cookie in request for captcha
	Cookie helper.Arguments
	// Pool is config for http clients pool
	Pool *httpclient.Config
}

func (c *Config) SetDefault() {
	c.Pool.SetDefault()

	if c.TokenTTL == 0 {
		c.TokenTTL = defaultTokenTTL
	}
}

func (c *Config) Validate() error {
	if c.VerifyURL == "" {
		return errors.EmptyConfigParameter("verifyurl")
	}

	if c.VerifyURL == "" {
		return errors.EmptyConfigParameter("key")
	}

	if c.TokenSign == "" {
		return errors.EmptyConfigParameter("tokensign")
	}

	return nil
}
