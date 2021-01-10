package cmd

import (
	"fmt"
	"os"

	"github.com/soldatov-s/accp/internal/admin"
	"github.com/soldatov-s/accp/internal/cache/external"
	"github.com/soldatov-s/accp/internal/cfg"
	context "github.com/soldatov-s/accp/internal/ctx"
	"github.com/soldatov-s/accp/internal/httpproxy"
	"github.com/soldatov-s/accp/internal/introspection"
	"github.com/soldatov-s/accp/internal/publisher"
	"github.com/soldatov-s/accp/internal/rabbitmq"
	externalcache "github.com/soldatov-s/accp/internal/redis"
	"github.com/spf13/cobra"
)

type empty struct{}

func serveHandler(command *cobra.Command, _ []string) {
	var err error

	// Create app context
	ctx := context.NewContext()

	// Fill appinfo
	ctx.FillAppInfo(appName, builded, hash, version, description)

	// Initilize config
	config, err := cfg.NewConfig(command)
	if err != nil {
		fmt.Println("failed to load config")
		os.Exit(0)
	}

	// Initilize logger
	ctx.InitilizeLogger(config.Logger)
	log := ctx.GetPackageLogger(empty{})

	log.Info().Msgf("Starting %s (%s)...", ctx.AppInfo.Name, ctx.AppInfo.GetBuildInfo())
	log.Info().Msg(ctx.AppInfo.Description)

	// Initilize introspector
	introspector, err := introspection.NewIntrospector(ctx, config.Introspector)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create introspector")
	}

	// Initilize external storage
	var externalStorage external.Storage
	if config.Redis != nil {
		externalStorage, err = externalcache.NewRedisClient(ctx, config.Redis)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create external storage")
		}
	}

	// Initilize pub
	var pub publisher.Publisher
	if config.Rabbitmq != nil {
		pub, err = rabbitmq.NewPublisher(ctx, config.Rabbitmq)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create publisher")
		}
	}

	// Initilize proxy
	proxy, err := httpproxy.NewHTTPProxy(ctx, config.Proxy, introspector, externalStorage, pub)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create proxy")
	}

	// Initilize admin
	adminsrv, err := admin.NewAdmin(ctx, config.Admin)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create admin")
	}

	// Start proxy
	go proxy.Start()

	// Start admin
	go adminsrv.Start()

	shutdown := func() {
		if err := proxy.Shutdown(); err != nil {
			log.Fatal().Err(err).Msg("failed to shutdow proxy")
		}

		if err := adminsrv.Shutdown(); err != nil {
			log.Fatal().Err(err).Msg("failed to shutdow admin")
		}
	}

	ctx.AppLoop(shutdown)
}
