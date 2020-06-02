package runner_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/hemantjadon/runner"
)

type job struct {
}

func (j job) Run(ctx context.Context) {
	select {
	case <-ctx.Done():
		time.Sleep(10 * time.Second)
	}
}

func ExampleRunner() {
	// Create a runner wrapping the job you want to execute.
	jobRunner := runner.Runner{Runnable: job{}}

	// Run the runner in a go routine.
	go func() {
		if err := jobRunner.Run(); err != nil && !errors.Is(err, runner.ErrClosed) {
			_, _ = fmt.Fprintf(os.Stderr, "error running runner: %v\n", err)
		}
	}()

	// Wait till the job is to be stopped.
	time.Sleep(5 * time.Second)

	// Then Close the runner, which will block till the runner completely stops.
	if err := jobRunner.Close(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error stopping runner: %v\n", err)
	}
}
