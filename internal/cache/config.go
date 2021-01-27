package cache

import (
	"github.com/soldatov-s/accp/internal/cache/external"
	"github.com/soldatov-s/accp/internal/cache/memory"
)

type Config struct {
	// Disabled is flag that cache disabled
	Disabled bool
	// Memory is a inmemory cache config
	Memory *memory.Config
	// External is a external cache config
	External *external.Config
}

func (c *Config) SetDefault() {
	if c.Disabled {
		return
	}

	if c.Memory == nil {
		c.Memory = &memory.Config{}
	}

	c.Memory.SetDefault()

	if c.External == nil {
		return
	}
	c.External.SetDefault()
}

func (c *Config) Merge(target *Config) *Config {
	if c == nil {
		return target
	}

	result := &Config{
		Disabled: c.Disabled,
		Memory:   c.Memory,
		External: c.External,
	}

	if target == nil {
		return result
	}

	if target.Disabled {
		result.Disabled = true
		return result
	}

	if !target.Disabled {
		result.Disabled = false
	}

	if target.Memory != nil {
		result.Memory = c.Memory.Merge(target.Memory)
	}

	if target.External != nil {
		result.External = c.External.Merge(target.External)
	}

	return result
}
