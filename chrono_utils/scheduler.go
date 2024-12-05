package chrono_utils

import (
	"codnect.io/chrono"
	"context"
)

type ContextTaskRunner struct {
	ctx context.Context
}

func NewContextTaskRunner(ctx context.Context) chrono.TaskRunner {
	if ctx == nil {
		ctx = context.Background()
	}

	return &ContextTaskRunner{ctx: ctx}
}

func (runner *ContextTaskRunner) Run(task chrono.Task) {
	go func() {
		task(runner.ctx)
	}()
}

// ContextTaskExecutor is a TaskExecutor that actually cancels the context passed to the executed tasks.
type ContextTaskExecutor struct {
	chrono.SimpleTaskExecutor
	cancel func()
}

func NewContextTaskExecutor() chrono.TaskExecutor {
	ctx, cancel := context.WithCancel(context.Background())

	return &ContextTaskExecutor{
		SimpleTaskExecutor: *chrono.NewSimpleTaskExecutor(NewContextTaskRunner(ctx)),
		cancel:             cancel,
	}
}

func (e *ContextTaskExecutor) Shutdown() chan bool {
	e.cancel()
	return e.SimpleTaskExecutor.Shutdown()
}

func NewContextTaskScheduler() chrono.TaskScheduler {
	return chrono.NewSimpleTaskScheduler(NewContextTaskExecutor())
}
