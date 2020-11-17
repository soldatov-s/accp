package introspection

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/rs/zerolog"
	context "github.com/soldatov-s/accp/internal/ctx"
	"github.com/soldatov-s/accp/internal/httpclient"
)

const (
	MaxHTTPBodySize = 1 << 20
)

type empty struct{}

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
	// PoolSize - size of pool httpclients for introspection requests
	PoolSize int
	// PoolTimeout - timeout of httpclients for introspection requests
	PoolTimeout time.Duration
}

type Introspector interface {
	IntrospectRequest(r *http.Request) ([]byte, error)
}

type Introspect struct {
	ctx          *context.Context
	cfg          *Config
	log          zerolog.Logger
	bodyTmpl     *template.Template
	pool         *httpclient.Pool
	reTrimFields *regexp.Regexp
}

// NewIntrospector creates Intrcopector
func NewIntrospector(ctx *context.Context, cfg *Config) (*Introspect, error) {
	i := &Introspect{ctx: ctx, cfg: cfg}

	i.pool = httpclient.NewPool(i.cfg.PoolSize, i.cfg.PoolTimeout)
	var err error
	i.bodyTmpl, err = template.New("body").Parse(i.cfg.BodyTemplate)
	if err != nil {
		return nil, err
	}

	i.log = ctx.GetPackageLogger(empty{})

	return i, nil
}

func (i *Introspect) initRegex() {
	filds := strings.Join(i.cfg.TrimmedFilds, "|")
	i.reTrimFields = regexp.MustCompile(`"(` + filds + `)":\s*("((\\"|[^"])*)"|\d*),?`)
}

// extractToken extract token from request
func (i *Introspect) extractToken(r *http.Request) (string, error) {
	if r == nil {
		return "", ErrBadAuthRequest
	}

	// Get token from query
	if r.URL == nil {
		return "", ErrBadAuthRequest
	}
	queryValues := r.URL.Query()
	for _, queryParamName := range i.cfg.QueryParamName {
		token := queryValues.Get(queryParamName)

		if token != "" {
			return token, nil
		}
	}

	// Get token from cookie
	for _, cookieName := range i.cfg.CookieName {
		tokenCookie, err := r.Cookie(cookieName)
		var token string
		if err == nil {
			token = tokenCookie.Value
		}

		if token != "" {
			return token, nil
		}
	}

	// If not token not found in query, try get from Authorization header
	token := r.Header.Get("Authorization")

	splitToken := strings.Split(token, " ")
	if len(splitToken) < 2 {
		return "", ErrBadAuthRequest
	}

	switch strings.ToLower(strings.TrimSpace(splitToken[0])) {
	case "bearer", "token":
		token = strings.TrimSpace(splitToken[1])
		return token, nil
	}

	return "", ErrBadAuthRequest
}

func (i *Introspect) trimFields(content []byte) []byte {
	return []byte(strings.ReplaceAll(i.reTrimFields.ReplaceAllString(string(content), ""), `,}`, "}"))
}

func (i *Introspect) IntrospectRequest(r *http.Request) ([]byte, error) {
	token, err := i.extractToken(r)
	if err != nil {
		return nil, err
	}

	if len(token) > 8 {
		i.log.Debug().Msgf("token from request: \"%s\"", token[0:4]+"****"+token[len(token)-4:])
	} else {
		i.log.Debug().Msgf("token from request: \"%s\"", token)
	}

	client := i.pool.GetFromPool()
	defer i.pool.PutToPool(client)

	req, err := i.buildRequest(token)
	if err != nil {
		return nil, err
	}

	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	contents, err := ioutil.ReadAll(io.LimitReader(response.Body, MaxHTTPBodySize))
	if err != nil {
		return nil, err
	}

	i.log.Debug().Msgf("introspection result: %s", string(contents))

	if !i.isValid(contents) {
		return nil, &ErrTokenInactive{token: token}
	}

	return i.trimFields(contents), nil
}

type tokenStruct struct {
	Token string
}

func (i *Introspect) buildRequest(token string) (*http.Request, error) {
	var request bytes.Buffer
	err := i.bodyTmpl.Execute(&request, tokenStruct{Token: token})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(i.cfg.Method, i.cfg.DSN+i.cfg.Endpoint, &request)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", i.cfg.ContentType)

	return req, nil
}

func (i *Introspect) isValid(contents []byte) bool {
	return strings.Contains(string(contents), i.cfg.ValidMarker)
}
