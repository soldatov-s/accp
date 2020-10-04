package cfg

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/soldatov-s/accp/internal/httpproxy"
	"github.com/soldatov-s/accp/internal/introspector"
	"github.com/soldatov-s/accp/internal/logger"
	"github.com/soldatov-s/accp/internal/rabbitmq"
	externalcache "github.com/soldatov-s/accp/internal/redis"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Configuration struct {
	Logger       *logger.LoggerConfig
	Proxy        *httpproxy.HTTPProxyConfig
	Introspector *introspector.IntrospectorConfig
	Redis        *externalcache.RedisConfig
	Rabbitmq     *rabbitmq.PublisherConfig
}

func NewConfig(command *cobra.Command) (*Configuration, error) {
	c := &Configuration{}
	if err := c.initialize(command); err != nil {
		return nil, err
	}
	return c, nil
}

// Initialize initializes configuration control structure and ensures that
// it is ready for working with configuration.
func (cfg *Configuration) initialize(command *cobra.Command) error {
	configPath, err := command.Flags().GetString("config")
	if err != nil {
		return err
	}

	configPath, configName := path.Split(configPath)
	configName = strings.TrimSuffix(configName, filepath.Ext(configName))

	viper.AddConfigPath(configPath)
	viper.SetConfigName(configName)
	viper.SetConfigType("yaml")

	err = viper.ReadInConfig() // Find and read the config file
	if err != nil {            // Handle errors reading the config file
		return err
	}

	if err := viper.UnmarshalKey("logger", &cfg.Logger); err != nil {
		return err
	}

	if err := viper.UnmarshalKey("introspector", &cfg.Introspector); err != nil {
		return err
	}

	if err := viper.UnmarshalKey("proxy", &cfg.Proxy); err != nil {
		return err
	}

	if err := viper.UnmarshalKey("redis", &cfg.Redis); err != nil {
		return err
	}

	if err := viper.UnmarshalKey("rabbitmq", &cfg.Rabbitmq); err != nil {
		return err
	}

	return nil
}
