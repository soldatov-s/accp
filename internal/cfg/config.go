package cfg

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/soldatov-s/accp/internal/admin"
	"github.com/soldatov-s/accp/internal/httpproxy"
	"github.com/soldatov-s/accp/internal/introspection"
	"github.com/soldatov-s/accp/internal/logger"
	"github.com/soldatov-s/accp/internal/rabbitmq"
	"github.com/soldatov-s/accp/internal/redis"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	DefaultConfigPath = "/etc/accp/config.yml"
)

type Config struct {
	Logger       *logger.Config
	Admin        *admin.Config
	Proxy        *httpproxy.Config
	Introspector *introspection.Config
	Redis        *redis.Config
	Rabbitmq     *rabbitmq.Config
}

func NewConfig(command *cobra.Command) (*Config, error) {
	configPath, err := command.Flags().GetString("config")
	if err != nil {
		return nil, err
	}

	if configPath == "" {
		configPath = DefaultConfigPath
	}

	configPath, configName := path.Split(configPath)
	configName = strings.TrimSuffix(configName, filepath.Ext(configName))

	viper.AddConfigPath(configPath)
	viper.SetConfigName(configName)
	viper.SetConfigType("yaml")

	err = viper.ReadInConfig() // Find and read the config file
	if err != nil {            // Handle errors reading the config file
		return nil, err
	}

	c := &Config{}
	if err := viper.Unmarshal(c); err != nil {
		return nil, err
	}
	return c, nil
}
