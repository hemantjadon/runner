# runner

`runner` provides an idiomatic wrapper for managing long-running tasks with the 
requirement of single atomic runs and graceful atomic shutdowns.

## Usage

Runner can be made by wrapping any `runnable` into the `Runner` type. `runnable`
can be any type satisfying the interface.

```go
type runnable interface {
    Run(ctx context.Context)    
}
```

The `runnable` must respect the `ctx` and should terminate when `ctx` completes.

Runner simply has 2 API methods `Run()` and `Close()`.

```go
type job struct {
}

func (j job) Run(ctx context.Context) {
	select {
	case <-ctx.Done():
		time.Sleep(10 * time.Second)
	}
}

func main() {
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
```

`Run()` starts the runnable job with a `ctx` provided, whereas `Close()` cancels
the `ctx` thereby shutting down the job.

## Can Run() and Close() be used concurrently?

Yes, `Run()` and `Close()` can be called concurrently, but the actual run and
close of the job will be atomic (the main goal of the package). If called
concurrently, only one of the calls to `Run()` and `Close()` will actually have
impact, rest all the other calls will exit with `ErrRunning` and `ErrStopping`
respectively.
