package httpproxy

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/accp/internal/cache/memory"
	ctxint "github.com/soldatov-s/accp/internal/ctx"
	"github.com/soldatov-s/accp/internal/httputils"
	"github.com/soldatov-s/accp/internal/introspector"
	accpmodels "github.com/soldatov-s/accp/models"
)

type empty struct{}

type Limit struct {
	Counter int
	Time    time.Duration
}

type Base struct {
	DSN        string
	TTL        time.Duration
	Tokenlimit *Limit
}

type Route struct {
	Base
}

type Service struct {
	Base
	Routes map[string]*Route
}

type HTTPProxyConfig struct {
	Listen    string
	Hydration struct {
		RequestID  bool
		Introspect string
	}
	Routes   map[string]*Route
	Services map[string]*Service
}

type HTTPProxy struct {
	cfg            *HTTPProxyConfig
	ctx            *ctxint.Context
	log            zerolog.Logger
	srv            *http.Server
	waitAnswerList map[string]chan struct{}
	waiteAnswerMu  map[string]*sync.Mutex
	memcache       memory.Cache
	introspector   introspector.Introspector
}

func NewHTTPProxy(ctx *ctxint.Context, cfg *HTTPProxyConfig, i introspector.Introspector) *HTTPProxy {
	p := &HTTPProxy{
		ctx:            ctx,
		cfg:            cfg,
		introspector:   i,
		waitAnswerList: make(map[string]chan struct{}),
		waiteAnswerMu:  make(map[string]*sync.Mutex),
	}

	p.srv = &http.Server{
		Addr:           cfg.Listen,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	p.srv.Handler = http.HandlerFunc(p.proxyHandler)

	p.log = ctx.GetPackageLogger(empty{})

	return p
}

func (p *HTTPProxy) waitAnswer(w http.ResponseWriter, hk string, ch chan struct{}) {
	<-ch

	if data, err := p.memcache.Select(hk); err == nil {
		if err := data.(*accpmodels.ResponseData).Write(w); err != nil {
			p.log.Err(err).Msg("failed to write data from cache")
		}
		return
	}

	http.Error(w, "failed to get data from cache", http.StatusServiceUnavailable)
}

func (p *HTTPProxy) getHandler(w http.ResponseWriter, r *http.Request) {
	// Finding a response to a request in the memory cache
	hk := hashKey(r)
	if data, err := p.memcache.Select(hk); err == nil {
		if err := data.(*accpmodels.ResponseData).Write(w); err != nil {
			p.log.Err(err).Msg("failed to write data from cache")
		}
		return
	}

	// Check that we not started to handle the request
	if waitCh, ok := p.waitAnswerList[hk]; !ok {
		// If we not started to handle the request we need to add lock-channel to map
		var (
			mu *sync.Mutex
			ok bool
		)
		// Create mutex for same requests
		if mu, ok = p.waiteAnswerMu[hk]; !ok {
			mu = &sync.Mutex{}
			p.waiteAnswerMu[hk] = mu
		}
		mu.Lock()
		if waitCh1, ok1 := p.waitAnswerList[hk]; !ok1 {
			ch := make(chan struct{})
			p.waitAnswerList[hk] = ch
			mu.Unlock() // unlock mutex fast as possible

			// Proxy request to backend
			resp, err := http.DefaultTransport.RoundTrip(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
			defer resp.Body.Close()

			responseData := &accpmodels.ResponseData{}
			if err := responseData.Read(resp); err != nil {
				p.log.Err(err).Msg("failed to read data from response")
			}

			if err := responseData.Write(w); err != nil {
				p.log.Err(err).Msg("failed to write data from response")
			}

			// Save answer to mem cache
			if err := p.memcache.Add(hk, responseData); err != nil {
				p.log.Err(err).Msg("failed to save data memcache")
			}

			close(ch)
			delete(p.waitAnswerList, hk)
			// Delete removes only item from map, GC remove mutex after removed all references to it.
			delete(p.waiteAnswerMu, hk)
		} else {
			mu.Unlock()
			p.waitAnswer(w, hk, waitCh1)
		}
	} else {
		p.waitAnswer(w, hk, waitCh)
	}
}

func (p *HTTPProxy) defaultHandler(w http.ResponseWriter, r *http.Request) {
	// Proxy request to backend
	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	httputils.CopyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
}

func (p *HTTPProxy) proxyHandler(w http.ResponseWriter, r *http.Request) {
	// Adding to request header the requestID
	p.hydrationID(r)

	// The check an authorization token
	if p.introspector != nil {
		introspectBody, err := p.introspector.IntrospectRequest(r)
		if _, ok := err.(*introspector.ErrTokenInactive); ok {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}

		p.hydrationIntrospect(r, introspectBody)
	}

	switch r.Method {
	case "GET":
		p.getHandler(w, r)
	default:
		p.defaultHandler(w, r)
	}
}

func hashKey(r *http.Request) string {
	sum := sha256.New().Sum([]byte(r.URL.RequestURI()))
	return base64.URLEncoding.EncodeToString(sum)
}

func (p *HTTPProxy) hydrationID(r *http.Request) {
	if !p.cfg.Hydration.RequestID {
		return
	}

	requestID := r.Header.Get("x-request-id")
	if requestID == "" {
		newUUID, err := uuid.NewRandom()
		if err != nil {
			p.log.Err(err).Msg("failed to generate requesID")
			return
		}
		r.Header.Add("x-request-id", newUUID.String())
	}
}

func (p *HTTPProxy) hydrationIntrospect(r *http.Request, content []byte) {
	var str string
	switch p.cfg.Hydration.Introspect {
	case "nothing":
		return
	case "plaintext":
		str = strings.ReplaceAll(strings.ReplaceAll(string(content), "\"", "\\\""), "\n", "")
	case "base64":
		str = base64.StdEncoding.EncodeToString(content)
	}

	r.Header.Add("accp-introspect-body", str)
	p.log.Debug().Msgf("accp-introspect-body header: %s", r.Header.Get("accp-introspect-body"))
}

func (p *HTTPProxy) Start() {
	p.log.Debug().Msg("start proxy")
	p.log.Fatal().Err(p.srv.ListenAndServe()).Msg("failed to start proxy")
}

func (p *HTTPProxy) Shutdown() error {
	return p.srv.Shutdown(context.Background())
}
