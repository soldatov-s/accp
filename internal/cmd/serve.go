package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/soldatov-s/accp/internal/app"
	"github.com/soldatov-s/accp/internal/captcha"
	"github.com/soldatov-s/accp/internal/cfg"
	"github.com/soldatov-s/accp/internal/httpproxy"
	"github.com/soldatov-s/accp/internal/introspection"
	"github.com/soldatov-s/accp/internal/logger"
	"github.com/soldatov-s/accp/internal/meta"
	"github.com/soldatov-s/accp/internal/rabbitmq"
	"github.com/soldatov-s/accp/internal/redis"
	"github.com/spf13/cobra"
)

type empty struct{}

func serveHandler(command *cobra.Command, _ []string) {
	// Create context
	ctx := context.Background()

	// Set app info
	ctx = meta.SetAppInfo(ctx, appName, builded, hash, version, description)

	// Load and parse config
	ctx, err := cfg.RegistrateAndParse(ctx, command)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("configuration parsed successfully")
	c := cfg.Get(ctx)

	// Registrate logger
	ctx = logger.RegistrateAndInitilize(ctx, c.Logger)

	// Get logger for package
	log := logger.GetPackageLogger(ctx, empty{})

	a := meta.Get(ctx)
	log.Info().Msgf("starting %s (%s)...", a.Name, a.GetBuildInfo())
	log.Info().Msg(a.Description)

	ctx, err = introspection.Registrate(ctx, c.Introspector)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to registrate introspection")
	}

	ctx, err = captcha.Registrate(ctx, c.Captcha)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to registrate captcha")
	}

	ctx, err = redis.Registrate(ctx, c.Redis)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to registrate redis")
	}

	ctx, err = rabbitmq.Registrate(ctx, c.Rabbitmq)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to registrate rabbitmq")
	}

	ctx, err = httpproxy.Registrate(ctx, c.Proxy)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to registrate proxy")
	}

	if err := app.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to start providers")
	}

	app.Loop(ctx)
}
