package runner

import (
	"context"

	"github.com/2comjie/ntool/util/gopool"
)

var RunTask func(ctx context.Context, f func())

func goRunTask(ctx context.Context, f func()) {
	go f()
}

// UseGoRunTask switches RunTask to run every task in a new goroutine.
func UseGoRunTask() {
	RunTask = goRunTask
}

func init() {
	RunTask = gopool.CtxGo
}
