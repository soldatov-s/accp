package captcha

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/accp/internal/httpclient"
	"github.com/soldatov-s/accp/internal/logger"
)

type SrcCaptcha int

const (
	defaultCaptchaType = "Accp-Captcha"
)

const (
	CaptchaSrcUnknown SrcCaptcha = iota
	CaptchaFromClient
	CaptchaFromGoogle
)

type empty struct{}

type AccpClaimsExample struct {
	*jwt.StandardClaims
	TokenType string
}

type GoogleCaptcha struct {
	ctx  context.Context
	cfg  *Config
	log  zerolog.Logger
	pool *httpclient.Pool
}

type GoogleRecaptchaResponse struct {
	Success            bool     `json:"success"`
	ChallengeTimestamp string   `json:"challenge_ts"`
	Hostname           string   `json:"hostname"`
	ErrorCodes         []string `json:"error-codes"`
}

func NewGoogleCaptcha(ctx context.Context, cfg *Config) (*GoogleCaptcha, error) {
	if cfg == nil {
		return nil, nil
	}

	cfg.SetDefault()

	err := cfg.Validate()
	if err != nil {
		return nil, err
	}

	g := &GoogleCaptcha{
		ctx:  ctx,
		cfg:  cfg,
		log:  logger.GetPackageLogger(ctx, empty{}),
		pool: httpclient.NewPool(cfg.Pool),
	}

	return g, nil
}

func (g *GoogleCaptcha) getCaptchaJWTToken(r *http.Request) string {
	for _, vv := range g.cfg.Cookie {
		if c, err := r.Cookie(vv); err == nil {
			return strings.TrimSpace(c.Value)
		}
	}

	for _, vv := range g.cfg.Header {
		if h := r.Header.Get(vv); h != "" {
			return strings.TrimSpace(h)
		}
	}

	return ""
}

func (g *GoogleCaptcha) validateReCAPTCHA(r *http.Request) error {
	client := g.pool.GetFromPool()
	defer g.pool.PutToPool(client)

	resp, err := client.PostForm(g.cfg.VerifyURL, url.Values{
		"secret":   {g.cfg.TokenSign},
		"response": {r.FormValue("g-recaptcha-response")},
	})

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var googleResponse GoogleRecaptchaResponse
	err = json.Unmarshal(body, &googleResponse)
	if err != nil {
		return err
	}
	if !googleResponse.Success {
		return ErrCaptchaFailed
	}

	return nil
}

func (g *GoogleCaptcha) validateCaptchaJWT(r *http.Request) error {
	jwtToken := g.getCaptchaJWTToken(r)

	if jwtToken == "" {
		return ErrCaptchaRequired
	}

	token, err := jwt.ParseWithClaims(
		jwtToken,
		&AccpClaimsExample{},
		func(token *jwt.Token) (interface{}, error) {
			// since we only use the one private key to sign the tokens,
			// we also only use its public counter part to verify
			return []byte(g.cfg.TokenSign), nil
		})

	if err != nil {
		return ErrCaptchaFailed
	}

	if claims, ok := token.Claims.(*AccpClaimsExample); !ok && !token.Valid {
		return ErrCaptchaFailed
	} else if claims.TokenType != defaultCaptchaType || claims.ExpiresAt < time.Now().Unix() {
		return ErrCaptchaFailed
	}

	return nil
}

func (g *GoogleCaptcha) Validate(r *http.Request) (SrcCaptcha, error) {
	if err := g.validateCaptchaJWT(r); err != nil {
		return CaptchaFromGoogle, g.validateReCAPTCHA(r)
	}

	return CaptchaFromClient, nil
}

func (g *GoogleCaptcha) GenerateCaptchaJWT(w http.ResponseWriter) error {
	iat := time.Now()
	expiration := iat.Add(g.cfg.TokenTTL)

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, &AccpClaimsExample{
		&jwt.StandardClaims{
			ExpiresAt: expiration.Unix(),
			IssuedAt:  iat.Unix(),
		},
		defaultCaptchaType,
	})

	// Creat token string
	jwtToken, err := t.SignedString([]byte(g.cfg.TokenSign))
	if err != nil {
		return err
	}

	for _, vv := range g.cfg.Cookie {
		cookie := http.Cookie{Name: vv, Value: jwtToken, Expires: expiration, HttpOnly: true}
		http.SetCookie(w, &cookie)
	}

	for _, vv := range g.cfg.Header {
		w.Header().Add(vv, jwtToken)
	}

	return nil
}
