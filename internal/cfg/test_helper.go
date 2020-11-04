package cfg

import (
	"bytes"

	"github.com/spf13/viper"
)

const (
	loggerConfig = `
logger:
  nocoloredoutput: true
  withtrace: false
  level: debug
`

	intropsectorConfig = `
introspector:
  dsn: http://localhost:8001
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

	proxyConfig = `proxy:
  listen: 0.0.0.0:9000
  hydration:
    requestid: true
    introspect: plaintext # add result of introspection as plaintext in header
  routes:
    /api/v1/:
      parameters:
        introspect: true
        dsn: http://192.168.100.48:30637
        pool:
          size: 20
          timeout: 10s
        routkey: V1 # rabbitmq rouykey
      routes:
        # subroutes
        users:
          parameters:
            limits:
              token:
                counter: 1000
                pt: 1m
              ip:
                counter: 1000
                pt: 1m
              deviceid:
                header: [device-id]
                cookie: [device-id]
                counter: 1000
                pt: 1m # cache auto update period
            refresh:
              count: 100
              time: 2m
            cache:
              memory:
                ttl: 30s
              external:
                keyprefix: users_
                ttl: 60s
  excluded:
    /api/v2/:
`
)

// /api/v1/users:
// parameters:
//   dsn: http://192.168.100.48:30637
//   pool:
//     size: 20
//     timeout: 10s
//   routkey: USERS
//   limits:
//     token:
//       counter: 1000
//       pt: 1m
//     ip:
//       counter: 1000
//       pt: 1m
//     deviceid:
//       header: device-id
//       cookie: device-id
//       counter: 1000
//       pt: 1m
//   refresh:
//     count: 100
//     time: 2m
//   cache:
//     memoryttl: 30s
//     externalttl: 60s

func LoadTestYAML() error {
	viper.SetConfigType("yaml")

	if err := viper.MergeConfig(bytes.NewBuffer([]byte(loggerConfig))); err != nil {
		return err
	}

	if err := viper.MergeConfig(bytes.NewBuffer([]byte(intropsectorConfig))); err != nil {
		return err
	}

	if err := viper.MergeConfig(bytes.NewBuffer([]byte(proxyConfig))); err != nil {
		return err
	}

	return nil
}

func LoadTestConfig() (*Configuration, error) {
	cfg := &Configuration{}

	if err := LoadTestYAML(); err != nil {
		return nil, err
	}

	if err := cfg.parse(); err != nil {
		return nil, err
	}

	return cfg, nil
}
