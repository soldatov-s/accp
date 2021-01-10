package testctxhelper

import (
	"testing"

	context "github.com/soldatov-s/accp/internal/ctx"
	"github.com/soldatov-s/accp/internal/logger"
)

func InitTestCtx(t *testing.T) *context.Context {
	lc := &logger.Config{
		Level:           "debug",
		NoColoredOutput: true,
		WithTrace:       false,
	}

	ctx := context.NewContext()
	ctx.InitilizeLogger(lc)

	return ctx
}
