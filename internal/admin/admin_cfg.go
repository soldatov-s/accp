package admin

const (
	defaultListen = "0.0.0.0:9100"
)

type Config struct {
	Listen string
	Pprof  bool
}

func (c *Config) Validate() error {
	if c.Listen == "" {
		c.Listen = defaultListen
	}

	return nil
}
