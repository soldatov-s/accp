package testhelpers

import (
	"bytes"

	"github.com/soldatov-s/accp/internal/cfg"
	"github.com/soldatov-s/accp/internal/httpproxy"
	"github.com/soldatov-s/accp/internal/introspection"
	"github.com/soldatov-s/accp/internal/logger"
	"github.com/spf13/viper"
)

const (
	LoggerConfig = `
logger:
  nocoloredoutput: true
  withtrace: false
  level: debug
`
	IntrospectorHost   = `localhost:8001`
	IntrospectorDSN    = `http://` + IntrospectorHost
	IntropsectorConfig = `
introspector:
  dsn: ` + IntrospectorDSN + `
  endpoint: /oauth2/introspect
  contenttype: application/x-www-form-urlencoded
  method: POST
  validmarker: '"active":true'
  bodytemplate: "token_type_hint=access_token&token={{.Token}}"
  cookiename: ["access-token"]
  queryparamname: ["access_token"]
  poolsize: 50
  pooltimeout: 10s
`

	ProxyConfig = `proxy:
  listen: 0.0.0.0:9000
  hydration:
    requestid: true
    introspect: plaintext # add result of introspection as plaintext in header
  routes:
    /api/v1/:
      parameters:
        introspect: true
        dsn: http://localhost:9090
        pool:
          size: 20
          timeout: 10s
        routekey: V1 # rabbitmq routekey
      routes:
        # subroutes
        users:
          parameters:
            limits:
              token:
                counter: 1
                pt: 3s
              ip:
                counter: 1
                pt: 3s
              deviceid:
                header: [device-id]
                cookie: [device-id]
                counter: 1
                pt: 1m # cache auto update period
            refresh:
              count: 10
              time: 10s
            cache:
              memory:
                ttl: 2s
              external:
                keyprefix: users_
                ttl: 30s
          excluded:
            - search
    /api/v1/users:
      parameters:
        dsn: http://localhost:9091
        pool:
          size: 20
          timeout: 10s
        routkey: USERS
        limits:
          token:
            counter: 1000
            pt: 1m
          ip:
            counter: 1000
            pt: 1m
          deviceid:
            header: device-id
            cookie: device-id
            counter: 1000
            pt: 1m
        refresh:
          count: 100
          time: 2m
        cache:
          memory:
            ttl: 2s
          external:
            keyprefix: users_
            ttl: 30s
`
)

func LoadTestYAML() error {
	viper.SetConfigType("yaml")

	if err := viper.MergeConfig(bytes.NewBuffer([]byte(LoggerConfig))); err != nil {
		return err
	}

	if err := viper.MergeConfig(bytes.NewBuffer([]byte(IntropsectorConfig))); err != nil {
		return err
	}

	if err := viper.MergeConfig(bytes.NewBuffer([]byte(ProxyConfig))); err != nil {
		return err
	}

	return nil
}

func LoadTestConfigLogger() (*logger.Config, error) {
	var lc *logger.Config
	if err := viper.UnmarshalKey("logger", &lc); err != nil {
		return nil, err
	}

	return lc, nil
}

func LoadTestConfigIntrospector() (*introspection.Config, error) {
	var ic *introspection.Config
	if err := viper.UnmarshalKey("introspector", &ic); err != nil {
		return nil, err
	}

	return ic, nil
}

func LoadTestConfigProxy() (*httpproxy.Config, error) {
	var pc *httpproxy.Config
	if err := viper.UnmarshalKey("proxy", &pc); err != nil {
		return nil, err
	}

	return pc, nil
}

func LoadTestConfig() (*cfg.Configuration, error) {
	c := &cfg.Configuration{}

	if err := LoadTestYAML(); err != nil {
		return nil, err
	}

	if err := c.Parse(); err != nil {
		return nil, err
	}

	return c, nil
}
