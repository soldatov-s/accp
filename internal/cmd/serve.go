package cmd

import (
	"fmt"
	"os"

	"github.com/soldatov-s/accp/internal/cache/external"
	"github.com/soldatov-s/accp/internal/cfg"
	context "github.com/soldatov-s/accp/internal/ctx"
	"github.com/soldatov-s/accp/internal/httpproxy"
	"github.com/soldatov-s/accp/internal/introspector"
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
	introspector, err := introspector.NewIntrospector(ctx, config.Introspector)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create introspector")
	}

	// Initilize external storage
	var externalStorage external.ExternalStorage
	if config.Redis != nil {
		externalStorage, err = externalcache.NewRedisClient(ctx, config.Redis)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create external storage")
		}
	}

	// Initilize publisher
	var publisher publisher.Publisher
	if config.Rabbitmq != nil {
		publisher, err = rabbitmq.NewPublisher(ctx, config.Rabbitmq)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create publisher")
		}
	}

	// Initilize proxy
	proxy, err := httpproxy.NewHTTPProxy(ctx, config.Proxy, introspector, externalStorage, publisher)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create proxy")
	}

	// Start proxy
	go proxy.Start()

	shutdown := func() {
		if err := proxy.Shutdown(); err != nil {
			log.Fatal().Err(err).Msg("failed to shutdow proxy")
		}
	}

	ctx.AppLoop(shutdown)

	// r := http.NewServeMux()

	// // Register pprof handlers
	// r.HandleFunc("/debug/pprof/", pprof.Index)
	// r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	// r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	// r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	// r.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// go http.ListenAndServe(":8090", r)
}
