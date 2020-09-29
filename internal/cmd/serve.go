package cmd

import (
	"fmt"
	"os"

	"github.com/soldatov-s/accp/internal/cfg"
	context "github.com/soldatov-s/accp/internal/ctx"
	"github.com/soldatov-s/accp/internal/httpproxy"
	"github.com/soldatov-s/accp/internal/introspector"
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

	// Initilize proxy
	proxy := httpproxy.NewHTTPProxy(ctx, config.Proxy, introspector)

	// Start proxy
	go proxy.Start()

	shutdown := func() {
		proxy.Shutdown()
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
