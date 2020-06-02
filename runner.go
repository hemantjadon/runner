// Package runner provides an idiomatic wrapper for managing long running tasks
// with requirement of single atomic runs and graceful atomic shutdowns.
//
// Runner can be made by wrapping any runnable into the Runner type.
//
// A runnable can be any type satisfying the interface.
//
//	type runnable interface {
// 	    Run(ctx context.Context)
// 	}
//
// The runnable must respect the ctx and should terminate when ctx completes.
//
// Run() starts the runnable job with a ctx provided, whereas Close() cancels
// the ctx thereby shutting down the job.
//
// Run() and Close() can be called concurrently, but the actual run and close of
// the job will be atomic (the main goal of the package). If called
// concurrently, only one of the calls to Run() and Close() will actually have
// impact, rest all the other calls will exit with ErrRunning and ErrStopping
// respectively.
package runner

import (
	"context"
	"sync"
	"sync/atomic"
)

const (
	// ErrClosed is returned when the runner shuts down completely.
	ErrClosed rerr = "runner closed"

	// ErrNotRunning is returned when trying to Close a non running runner.
	ErrNotRunning rerr = "runner not running"

	// ErrRunning is returned when trying to Run an already running runner.
	ErrRunning rerr = "runner already running"

	// ErrStopping is returned when trying to Close an already stopping runner.
	ErrStopping rerr = "runner already stopping"
)

// Runner describes a runner which runs the provided runnable atomically once.
type Runner struct {
	// Runnable to be run by the runner.
	Runnable runnable

	running  uint32        // 0 if not running 1 if running
	stopping uint32        // 0 if not stopping 1 if stopping
	mu       sync.Mutex    // Serializes the access to below fields.
	stopCh   chan struct{} // Closed by Close to signal a close operation
	quitCh   chan struct{} // Closed by Run to signal successful shutdown
}

// Run start up the runner and blocks until the Close() is called or underlying
// Runnable's execution completes.
//
// Upon successful startup it blocks while the underlying Runnable is running
// with a ctx provided to control the execution of the job. When Close() is
// called the ctx completes signalling the underlying runnable to shutdown
// gracefully. If underlying runnable respects the ctx (which it should) then
// after the graceful shutdown it ErrClosed is returned.
//
// If trying to run runner which is already running then ErrRunning is returned.
//
// It runs the underlying runnable atomically. The runner can be safely
// restarted when it is shut down completely.
//
// Run is safe to be called concurrently by multiple go routines, but only one
// of them will actually run the underlying runnable, all the other calls will
// return with ErrRunning.
func (a *Runner) Run() error {
	if !atomic.CompareAndSwapUint32(&a.running, 0, 1) {
		return ErrRunning
	}
	defer atomic.StoreUint32(&a.running, 0)

	quitCh := make(chan struct{})
	defer func() { close(quitCh) }()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stopCh := make(chan struct{})
	go func() {
		select {
		case <-stopCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	a.mu.Lock()
	a.stopCh = stopCh
	a.quitCh = quitCh
	a.mu.Unlock()

	a.Runnable.Run(ctx)

	return ErrClosed
}

// Close shuts down the runner gracefully. It blocks until the runner is closed,
// then a nil error is returned if shutdown completes successfully.
//
// Close is safe to be called concurrently by multiple go routines, but only one
// of those calls will actually shut down the runner, rest all ofthem will
// return with ErrStopping.
//
// If Close is called on a non running runner then ErrNotRunning is returned.
func (a *Runner) Close() error {
	running := atomic.LoadUint32(&a.running)
	if running == 0 {
		return ErrNotRunning
	}
	if !atomic.CompareAndSwapUint32(&a.stopping, 0, 1) {
		return ErrStopping
	}
	defer atomic.StoreUint32(&a.stopping, 0)

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stopCh == nil || a.quitCh == nil {
		return ErrNotRunning
	}
	close(a.stopCh)
	<-a.quitCh
	return nil
}

type rerr string

func (e rerr) Error() string {
	return string(e)
}

type runnable interface {
	Run(ctx context.Context)
}
