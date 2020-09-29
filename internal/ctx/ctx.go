package context

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/soldatov-s/accp/internal/app"
	"github.com/soldatov-s/accp/internal/logger"
)

type Context struct {
	Logger *logger.Logger
	// ApplicationInfo about service
	AppInfo app.ApplicationInfo
}

// NewContext creates new Context structure. If structure already created
// then pointer to that structure will be returned. This function will call
// Context.initialize() on instantiation to ensure that basic Context things
// are initialized properly.
func NewContext() *Context {
	return &Context{}
}

func (c *Context) InitilizeLogger(cfg *logger.LoggerConfig) {
	c.Logger = logger.NewLogger(cfg)
}

// FillAppInfo fill application info about service
func (c *Context) FillAppInfo(name, builded, hash, version, description string) {
	c.AppInfo = app.ApplicationInfo{
		Name:        name,
		Builded:     builded,
		Hash:        hash,
		Version:     version,
		Description: description,
	}
}

// GetPackageLogger return logger for package
func (c *Context) GetPackageLogger(emptyStruct interface{}) zerolog.Logger {
	log := c.Logger.GetLogger(c.AppInfo.Name, nil)
	return logger.InitializeLogger(&log, emptyStruct)
}

// AppLoop is application loop, exit on SIGTERM
func (c *Context) AppLoop(shutdown func()) {
	var closeSignal chan os.Signal
	log := c.Logger.GetLogger(c.AppInfo.Name, nil)

	exit := make(chan struct{})
	closeSignal = make(chan os.Signal, 1)
	signal.Notify(closeSignal, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-closeSignal
		shutdown()
		log.Info().Msg("Exit program")
		close(exit)
	}()

	// Exit app if chan is closed
	<-exit
}
