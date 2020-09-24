package httpsrv

import (
	"bytes"
	"crypto/sha256"
	"encoding"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ory/hydra/sdk/go/hydra/models"
)

const (
	defaultClientTimeout = 60 * time.Second
	MaxHTTPBodySize      = 1 << 20

	introspectEndpoint = "/oauth2/introspect"
)

var (
	ErrBadAuthRequest  = errors.New("bad authorization request")
	ErrNotFoundInCache = errors.New("not found in cache")
	Storage            sync.Map
	poolHTTP           *HTTPClientPool
)

type HTTPClient struct {
	*http.Client
	busy bool
	id   int
	req  *http.Request
	buf  *bytes.Buffer
}

type HTTPClientPool struct {
	muPool  sync.Mutex
	clients []*HTTPClient
	freeCnt int
	ch      chan *HTTPClient
}

func NewHTTPClientPool(size int) *HTTPClientPool {
	p := &HTTPClientPool{}
	p.clients = make([]*HTTPClient, 0, size)
	for i := 0; i < size; i++ {

		buf := bytes.NewBufferString("token_type_hint=access_token&token=")
		req, _ := http.NewRequest("POST", URL, buf)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		c := &HTTPClient{Client: NewHTTPClient(0), id: i, req: req, buf: buf}
		p.clients = append(p.clients, c)
	}
	p.ch = make(chan *HTTPClient)
	go func() {
		p.ch <- p.clients[0]
		// log.Println("NewHTTPClientPool put client to pool")
	}()
	return p
}

func (p *HTTPClientPool) GetFromPool() *HTTPClient {
	p.muPool.Lock()
	defer p.muPool.Unlock()
	for c, ok := <-p.ch; ok; {
		go func() {
			for _, cc := range p.clients {
				if !cc.busy {
					p.ch <- cc
					// log.Println("GetFromPool put client to pool ", cc.id)
				}
			}
		}()
		// log.Println("GetFromPool get client from pull ", c.id)
		c.busy = true
		return c
	}

	return nil
}

type ErrTokenInactive struct {
	token string
}

type ResponseData struct {
	Body       string
	Header     http.Header
	StatusCode int
}

func (r *ResponseData) MarshalBinary() (data []byte, err error) {
	return json.Marshal(r)
}

func (r *ResponseData) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, r)
}

// CacheData is a data which putting in cache
type CacheData interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

// CacheItem is an item of cache
type CacheItem struct {
	Data      CacheData
	TimeStamp time.Time
	UUID      string
}

func Add(key string, data CacheData) error {
	if _, ok := Storage.Load(key); !ok {
		Storage.Store(key, &CacheItem{
			Data:      data,
			TimeStamp: time.Now().UTC(),
		})

		log.Printf("add key %s to cache", key)
	}
	return nil
}

func Select(key string) (CacheData, error) {
	if v, ok := Storage.Load(key); ok {
		dd := v.(*CacheItem)
		// log.Printf("select %s from cache", key)
		return dd.Data, nil
	}

	return nil, ErrNotFoundInCache
}

func (e *ErrTokenInactive) Error() string {
	return fmt.Sprintf("token %s inactive", e.token)
}

var netTransport *http.Transport

func NewHTTPClient(timeout time.Duration) *http.Client {
	clientTimeout := defaultClientTimeout
	if timeout > 0 {
		clientTimeout = timeout
	}

	if netTransport == nil {
		dialer := &net.Dialer{
			Timeout: clientTimeout,
		}

		netTransport = &http.Transport{
			MaxIdleConns:          1024,
			MaxIdleConnsPerHost:   1024,
			Dial:                  dialer.Dial,
			TLSHandshakeTimeout:   clientTimeout,
			ExpectContinueTimeout: clientTimeout,
			IdleConnTimeout:       clientTimeout,
			ResponseHeaderTimeout: clientTimeout,
		}
	}
	return &http.Client{
		Transport: netTransport,
		Timeout:   clientTimeout,
	}
}

// ExtractToken extract token from request
func ExtractToken(r *http.Request) (string, error) {
	if r == nil {
		return "", ErrBadAuthRequest
	}

	// Get token from query
	if r.URL == nil {
		return "", ErrBadAuthRequest
	}
	queryValues := r.URL.Query()
	token := queryValues.Get("access_token")

	if token != "" {
		return token, nil
	}

	// Get token from cookie
	tokenCookie, err := r.Cookie("wasd-access-token")
	if err == nil {
		token = tokenCookie.Value
	}

	if token != "" {
		return token, nil
	}

	// If not token not found in query, try get from Authorization header
	token = r.Header.Get("Authorization")

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

func IntrospectRequest(r *http.Request) (*models.Introspection, error) {
	token, err := ExtractToken(r)
	if err != nil {
		return nil, err
	}

	return introspectToken(token)
}

var URL = "http://192.168.100.48:30611" + introspectEndpoint

func introspectToken(token string) (*models.Introspection, error) {
	request := bytes.NewBufferString("token_type_hint=access_token&token=" + token)

	client := poolHTTP.GetFromPool()
	defer func() {
		client.busy = false
	}()
	req, err := http.NewRequest("POST", URL, request)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// n, err := client.buf.WriteString(token)
	// fmt.Println(n, err, client.buf.String())
	// request.WriteString(token)

	response, err := client.Do(req)
	if err != nil {
		// fmt.Println("err: ", err)
		// fmt.Println("response: ", response)
		return nil, err
	}
	defer response.Body.Close()

	contents, err := ioutil.ReadAll(io.LimitReader(response.Body, MaxHTTPBodySize))
	if err != nil {
		return nil, err
	}

	jData := &models.Introspection{}

	err = json.Unmarshal(contents, jData)
	if err != nil {
		return nil, err
	}

	// fmt.Printf("%+v/n", jData)

	if !*jData.Active {
		return nil, &ErrTokenInactive{token: token}
	}
	return jData, nil

}

func requestIDHydration(req *http.Request) {
	requestID := req.Header.Get("x-request-id")
	if requestID == "" {
		newUUID, err := uuid.NewRandom()
		if err != nil {
			log.Printf("failed to generate requesID: %s", err)
		}
		req.Header.Add("x-request-id", newUUID.String())
	}
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func handleHTTP(w http.ResponseWriter, req *http.Request) {
	_, err := IntrospectRequest(req)
	if _, ok := err.(*ErrTokenInactive); ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// bufReq := new(bytes.Buffer)
	// _, err = io.Copy(bufReq, req.Body)
	// log.Println("Request Body")
	// log.Println(bufReq.String())
	// log.Printf("Query %s", req.URL.RequestURI())

	sum := sha256.New().Sum([]byte(req.URL.RequestURI()))
	hasKey := base64.URLEncoding.EncodeToString(sum)

	if data, err := Select(hasKey); err == nil {
		// buf := new(bytes.Buffer)
		responseData := data.(*ResponseData)
		// buf.WriteString(responseData.Body)
		// log.Printf("response from cache %s", buf.String())
		copyHeader(w.Header(), responseData.Header)
		w.WriteHeader(responseData.StatusCode)
		w.Write([]byte(responseData.Body))
		return
	}

	requestIDHydration(req)
	// for k, vv := range req.Header {
	// 	for _, v := range vv {
	// 		log.Print(k, ": ", v)
	// 	}
	// }

	// log.Println(req.Method, req.Host)
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	buf := new(bytes.Buffer)
	io.Copy(buf, resp.Body)
	w.Write(buf.Bytes())

	responseData := &ResponseData{Body: buf.String()}
	responseData.Header = make(http.Header)
	copyHeader(responseData.Header, resp.Header)
	responseData.StatusCode = resp.StatusCode
	Add(hasKey, responseData)

	// log.Println("Response Body")
	// log.Println(buf.String())
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	handleHTTP(w, r)
}

func Start() {
	poolHTTP = NewHTTPClientPool(20)
	s := &http.Server{
		Addr:           ":8080",
		Handler:        http.HandlerFunc(proxyHandler),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}
