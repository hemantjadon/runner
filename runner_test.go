package runner

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/adamluzsi/testcase"
)

type blockingRunnable struct {
}

func (blockingRunnable) Run(ctx context.Context) {
	select {
	case <-ctx.Done():
	}
}

func TestRunnerIntegration(t *testing.T) {
	s := testcase.NewSpec(t)

	s.Describe("startup and shutdown", SpecStartupAndShutdown)
}

func SpecStartupAndShutdown(s *testcase.Spec) {
	s.Let(`runner`, func(t *testcase.T) interface{} {
		return &Runner{Runnable: &blockingRunnable{}}
	})

	s.When(`runner is not running`, func(s *testcase.Spec) {
		s.And(`it is closed`, func(s *testcase.Spec) {
			s.Let(`closeErrCh`, func(t *testcase.T) interface{} { return make(chan error) })
			s.Before(func(t *testcase.T) {
				agg := t.I(`runner`).(*Runner)
				closeErrCh := t.I(`closeErrCh`).(chan error)
				go func() {
					closeErrCh <- agg.Close()
				}()
			})

			s.Then(`error ErrNotRunning is returned`, func(t *testcase.T) {
				closeErrCh := t.I(`closeErrCh`).(chan error)
				err := <-closeErrCh
				if !errors.Is(err, ErrNotRunning) {
					t.Errorf("got err = %v, want err = %v", err, ErrNotRunning)
				}
			})
		})

		s.And(`it is closed concurrently`, func(s *testcase.Spec) {
			s.Let(`concurrentCalls`, func(t *testcase.T) interface{} { return int(10E3) })

			s.Let(`closeErrCh`, func(t *testcase.T) interface{} {
				concurrentCalls := t.I(`concurrentCalls`).(int)
				return make(chan error, concurrentCalls)
			})

			s.Before(func(t *testcase.T) {
				agg := t.I(`runner`).(*Runner)
				closeErrCh := t.I(`closeErrCh`).(chan error)
				concurrentCalls := t.I(`concurrentCalls`).(int)
				var wg sync.WaitGroup
				for i := 0; i < concurrentCalls; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						closeErrCh <- agg.Close()
					}()
				}
				go func() {
					wg.Wait()
					close(closeErrCh)
				}()
			})

			s.Then(`all calls return ErrNotRunning`, func(t *testcase.T) {
				closeErrCh := t.I(`closeErrCh`).(chan error)
				var fail bool
				for err := range closeErrCh {
					if !errors.Is(err, ErrNotRunning) {
						fail = true
					}
				}
				if fail {
					t.Errorf("some calls did not return ErrNotRunning")
				}
			})
		})
	})

	s.When(`runner is running`, func(s *testcase.Spec) {
		s.Let(`runErrCh`, func(t *testcase.T) interface{} { return make(chan error) })

		s.Before(func(t *testcase.T) {
			agg := t.I(`runner`).(*Runner)
			runErrCh := t.I(`runErrCh`).(chan error)
			go func() {
				runErrCh <- agg.Run()
				close(runErrCh)
			}()
			time.Sleep(1 * time.Millisecond) // To make sure that go routine runs
		})

		s.And(`it is run again`, func(s *testcase.Spec) {
			s.Let(`run2ErrCh`, func(t *testcase.T) interface{} { return make(chan error) })

			s.Before(func(t *testcase.T) {
				agg := t.I(`runner`).(*Runner)
				run2ErrCh := t.I(`run2ErrCh`).(chan error)
				go func() {
					run2ErrCh <- agg.Run()
				}()
			})

			s.Then(`error ErrRunning is returned`, func(t *testcase.T) {
				run2ErrCh := t.I(`run2ErrCh`).(chan error)
				err := <-run2ErrCh
				if !errors.Is(err, ErrRunning) {
					t.Errorf("got err = %v, want err = %v", err, ErrRunning)
				}
			})
		})

		s.And(`it is closed`, func(s *testcase.Spec) {
			s.Let(`closeErrCh`, func(t *testcase.T) interface{} { return make(chan error) })

			s.Before(func(t *testcase.T) {
				agg := t.I(`runner`).(*Runner)
				closeErrCh := t.I(`closeErrCh`).(chan error)
				go func() {
					closeErrCh <- agg.Close()
				}()
			})

			s.Then(`nil error is returned`, func(t *testcase.T) {
				closeErrCh := t.I(`closeErrCh`).(chan error)
				err := <-closeErrCh
				if err != nil {
					t.Errorf("got err = %v, want err = <nil>", err)
				}
			})

			s.Then(`run returns with ErrClosed`, func(t *testcase.T) {
				runErrCh := t.I(`runErrCh`).(chan error)
				runErr := <-runErrCh

				if !errors.Is(runErr, ErrClosed) {
					t.Errorf("got run err = %v, want run err = %v", runErr, ErrClosed)
				}
			})
		})

		s.And(`it is closed concurrently`, func(s *testcase.Spec) {
			s.Let(`concurrentCalls`, func(t *testcase.T) interface{} { return int(10E3) })

			s.Let(`closeErrCh`, func(t *testcase.T) interface{} {
				concurrentCalls := t.I(`concurrentCalls`).(int)
				return make(chan error, concurrentCalls)
			})

			s.Before(func(t *testcase.T) {
				agg := t.I(`runner`).(*Runner)
				closeErrCh := t.I(`closeErrCh`).(chan error)
				concurrentCalls := t.I(`concurrentCalls`).(int)
				var wg sync.WaitGroup
				for i := 0; i < concurrentCalls; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						closeErrCh <- agg.Close()
					}()
				}
				go func() {
					wg.Wait()
					close(closeErrCh)
				}()
			})

			s.Then(`one call return nil error while all other calls return ErrNotRunning while run returns with ErrClosed`, func(t *testcase.T) {
				closeErrCh := t.I(`closeErrCh`).(chan error)
				var errTotalCount int
				var errNotRunningOrStoppingCount int
				var errNilCount int
				for err := range closeErrCh {
					errTotalCount++
					if err == nil {
						errNilCount++
						continue
					}
					if errors.Is(err, ErrNotRunning) || errors.Is(err, ErrStopping) {
						errNotRunningOrStoppingCount++
						continue
					}
				}
				if errNilCount != 1 {
					t.Errorf("got <nil> errors = %v, want <nil> errors = %v", errNilCount, 1)
				}
				if errNotRunningOrStoppingCount != errTotalCount-1 {
					t.Errorf("got %v errors = %v, want %v errors = %v", ErrNotRunning, errNotRunningOrStoppingCount, ErrNotRunning, errTotalCount-1)
				}
			})

			s.Then(`run returns with ErrClosed`, func(t *testcase.T) {
				runErrCh := t.I(`runErrCh`).(chan error)
				runErr := <-runErrCh

				if !errors.Is(runErr, ErrClosed) {
					t.Errorf("got run err = %v, want run err = %v", runErr, ErrClosed)
				}
			})
		})
	})
}
