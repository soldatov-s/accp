package logger

import (
	// stdlib
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	// other
	"github.com/rs/zerolog"
)

const (
	LoggerLevelDebug    = "DEBUG"
	LoggerLevelInfo     = "INFO"
	LoggerLevelWarn     = "WARN"
	LoggerLevelError    = "ERROR"
	LoggerLevelFatal    = "FATAL"
	LoggerLevelPanic    = "PANIC"
	LoggerLevelDisabled = "DISABLED"
)

// Logger is a controlling structure for application's logger.
type Logger struct {
	// Main logger
	logger zerolog.Logger
	// Other created loggers.
	loggers      map[string]zerolog.Logger
	loggersMutex sync.RWMutex
	initialized  bool
}

// nolint
func loggerCtxByField(loggerCtx *zerolog.Context, field *Field) zerolog.Context {
	ctx := *loggerCtx
	switch field.Type {
	case reflect.Bool:
		ctx = ctx.Bool(field.Name, field.Value.(bool))
	case reflect.Float32:
		ctx = ctx.Float32(field.Name, field.Value.(float32))
	case reflect.Float64:
		ctx = loggerCtx.Float64(field.Name, field.Value.(float64))
	case reflect.Int:
		ctx = ctx.Int(field.Name, field.Value.(int))
	case reflect.Int8:
		ctx = ctx.Int8(field.Name, field.Value.(int8))
	case reflect.Int16:
		ctx = loggerCtx.Int16(field.Name, field.Value.(int16))
	case reflect.Int32:
		ctx = loggerCtx.Int32(field.Name, field.Value.(int32))
	case reflect.Int64:
		ctx = ctx.Int64(field.Name, field.Value.(int64))
	case reflect.Interface:
		ctx = ctx.Interface(field.Name, field.Value)
	case reflect.String:
		ctx = ctx.Str(field.Name, field.Value.(string))
	case reflect.Uint:
		ctx = ctx.Uint(field.Name, field.Value.(uint))
	case reflect.Uint8:
		ctx = ctx.Uint8(field.Name, field.Value.(uint8))
	case reflect.Uint16:
		ctx = ctx.Uint16(field.Name, field.Value.(uint16))
	case reflect.Uint32:
		ctx = ctx.Uint32(field.Name, field.Value.(uint32))
	case reflect.Uint64:
		ctx = ctx.Uint64(field.Name, field.Value.(uint64))
	}
	return ctx
}

// GetLogger creates new logger if not exists and fills it with defined fields.
// If requested logger already exists - it'll be returned.
func (l *Logger) GetLogger(name string, fields []*Field) zerolog.Logger {
	l.loggersMutex.RLock()
	logger, found := l.loggers[name]
	l.loggersMutex.RUnlock()

	if found {
		return logger
	}

	loggerCtx := l.logger.With()

	for _, field := range fields {
		loggerCtx = loggerCtxByField(&loggerCtx, field)
	}

	l.loggersMutex.Lock()
	l.loggers[name] = loggerCtx.Logger()
	l.loggersMutex.Unlock()

	return loggerCtx.Logger()
}

func NewLogger(cfg *Config) *Logger {
	l := &Logger{}

	// Initialize logger
	l.Initialize(
		cfg.Level,
		cfg.NoColoredOutput,
		cfg.WithTrace,
	)

	return l
}

func (l *Logger) Initialize(lvl string, noColor, withTrace bool) {
	switch strings.ToUpper(lvl) {
	case LoggerLevelDebug:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case LoggerLevelInfo:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case LoggerLevelWarn:
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case LoggerLevelError:
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case LoggerLevelFatal:
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case LoggerLevelPanic:
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	case LoggerLevelDisabled:
		zerolog.SetGlobalLevel(zerolog.Disabled)
	}

	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		NoColor:    noColor,
		TimeFormat: time.RFC3339,
	}

	output.FormatLevel = func(i interface{}) string {
		var v string

		if ii, ok := i.(string); ok {
			ii = strings.ToUpper(ii)
			switch ii {
			case LoggerLevelDebug, LoggerLevelError, LoggerLevelFatal,
				LoggerLevelInfo, LoggerLevelWarn, LoggerLevelPanic:
				v = fmt.Sprintf("%-5s", ii)
			default:
				v = ii
			}
		}

		return fmt.Sprintf("| %s |", v)
	}

	l.logger = zerolog.New(output).With().Timestamp().Logger()
	l.logger = l.logger.Hook(TracingHook{WithTrace: withTrace})

	l.loggers = make(map[string]zerolog.Logger)
	l.initialized = true
}

// IsInitialized returns true if logger was initialized and configured.
func (l *Logger) IsInitialized() bool {
	return l.initialized
}

// InitializeLogger initilizes fields for domains and packages
func InitializeLogger(parent *zerolog.Logger, emptyStruct interface{}) (log zerolog.Logger) {
	res := false
	packageTypes := []string{"domains", "internal"}
	packageType := ""
	pkgPath := reflect.TypeOf(emptyStruct).PkgPath()
	pathElements := strings.Split(pkgPath, "/")

	i := 0

	for _, pathElement := range pathElements {
		for _, packageType = range packageTypes {
			if pathElement == packageType {
				log = parent.With().Str("type", pathElements[i]).Logger()
				log = log.With().Str("package", pathElements[i+1]).Logger()
				res = true

				break
			}
		}

		if res {
			break
		}
		i++
	}

	if packageType == "domains" {
		ver, _ := strconv.ParseInt(strings.TrimPrefix(pathElements[i+2], "v"), 10, 64)
		log = log.With().Int64("version", ver).Logger()
		i++
	}

	if i+3 <= len(pathElements) {
		log = log.With().Str("subsystem", pathElements[i+2]).Logger()
	}

	log.Info().Msg("Initializing...")

	return log
}
