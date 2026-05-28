package ioc

import (
	"context"
	"errors"
	"testing"
	"time"
)

const circularDeadlockTimeout = 250 * time.Millisecond

type cycleCService interface {
	DoC() string
}

type _cycleCServiceImpl struct{}

func (*_cycleCServiceImpl) DoC() string {
	return "C"
}

func TestCrossGoroutineSelfDependencyPanicsCircular(t *testing.T) {
	r := runCircularGuarded(t, func() {
		var ctx context.Context
		ctx = WithProvided(context.Background(), func() *_ServeServiceImpl {
			if recovered := recoverFromGoroutine(func() {
				_ = GetProvided[ServeService](ctx)
			}); recovered != nil {
				panic(recovered)
			}
			return &_ServeServiceImpl{}
		})
		_ = GetProvided[ServeService](ctx)
	})
	expectCircularPanic(t, r)
}

func TestCrossGoroutineMutualDependencyPanicsCircular(t *testing.T) {
	r := runCircularGuarded(t, func() {
		var ctx context.Context
		ctx = WithProvided(context.Background(),
			func() *_ServeServiceImpl {
				if recovered := recoverFromGoroutine(func() {
					_ = GetProvided[DoSomethingService](ctx)
				}); recovered != nil {
					panic(recovered)
				}
				return &_ServeServiceImpl{}
			},
			func() *_DoSomethingServiceImpl {
				_ = GetProvided[ServeService](ctx)
				return &_DoSomethingServiceImpl{}
			},
		)
		_ = GetProvided[ServeService](ctx)
	})
	expectCircularPanic(t, r)
}

func TestCrossGoroutineLongDependencyCyclePanicsCircular(t *testing.T) {
	r := runCircularGuarded(t, func() {
		var ctx context.Context
		ctx = WithProvided(context.Background(),
			func() *_DoAServiceImpl {
				if recovered := recoverFromGoroutine(func() {
					_ = GetProvided[ServiceB](ctx)
				}); recovered != nil {
					panic(recovered)
				}
				return &_DoAServiceImpl{}
			},
			func() *_DoBServiceImpl {
				_ = GetProvided[cycleCService](ctx)
				return &_DoBServiceImpl{}
			},
			func() *_cycleCServiceImpl {
				_ = GetProvided[ServiceA](ctx)
				return &_cycleCServiceImpl{}
			},
		)
		_ = GetProvided[ServiceA](ctx)
	})
	expectCircularPanic(t, r)
}

func TestCrossGoroutineSelfDependencyViaIsProvidedPanicsCircular(t *testing.T) {
	r := runCircularGuarded(t, func() {
		var ctx context.Context
		ctx = WithProvided(context.Background(), func() *_ServeServiceImpl {
			if recovered := recoverFromGoroutine(func() {
				_ = IsProvided[ServeService](ctx)
			}); recovered != nil {
				panic(recovered)
			}
			return &_ServeServiceImpl{}
		})
		_ = GetProvided[ServeService](ctx)
	})
	expectCircularPanic(t, r)
}

func TestCrossGoroutineMutualDependencyViaOptionalPanicsCircular(t *testing.T) {
	r := runCircularGuarded(t, func() {
		var ctx context.Context
		ctx = WithProvided(context.Background(),
			func() *_ServeServiceImpl {
				if recovered := recoverFromGoroutine(func() {
					_, _ = GetOptionalProvided[DoSomethingService](ctx)
				}); recovered != nil {
					panic(recovered)
				}
				return &_ServeServiceImpl{}
			},
			func() *_DoSomethingServiceImpl {
				_ = GetProvided[ServeService](ctx)
				return &_DoSomethingServiceImpl{}
			},
		)
		_ = GetProvided[ServeService](ctx)
	})
	expectCircularPanic(t, r)
}

func TestSilencePanicsHandlesCrossGoroutineCircularDependency(t *testing.T) {
	r := runCircularGuarded(t, func() {
		var ctx context.Context
		ctx = SilencePanics(context.Background())
		ctx = WithProvided(ctx, func() *_ServeServiceImpl {
			if recovered := recoverFromGoroutine(func() {
				_ = GetProvided[ServeService](ctx)
			}); recovered != nil {
				panic(recovered)
			}
			return &_ServeServiceImpl{}
		})
		if value, found := GetOptionalProvided[ServeService](ctx); found || value != nil {
			t.Fatalf("expected silenced circular factory to report absent, got (%v,%v)", value, found)
		}
	})
	if r != nil {
		t.Fatalf("expected SilencePanics to handle CircularInjectable, got %v", r)
	}
}

func TestOnPanicHandlesCrossGoroutineCircularDependency(t *testing.T) {
	var handled int
	r := runCircularGuarded(t, func() {
		var ctx context.Context
		ctx = OnPanic(context.Background(), func(recovered any) bool {
			err, ok := recovered.(error)
			if ok && errors.Is(err, CircularInjectable) {
				handled++
				return true
			}
			return false
		})
		ctx = WithProvided(ctx, func() *_ServeServiceImpl {
			if recovered := recoverFromGoroutine(func() {
				_ = GetProvided[ServeService](ctx)
			}); recovered != nil {
				panic(recovered)
			}
			return &_ServeServiceImpl{}
		})
		_, found := GetOptionalProvided[ServeService](ctx)
		if found {
			t.Fatal("expected handled circular factory to report absent")
		}
	})
	if r != nil {
		t.Fatalf("expected OnPanic to handle CircularInjectable, got %v", r)
	}
	if handled != 1 {
		t.Fatalf("expected OnPanic handler to handle one circular panic, got %d", handled)
	}
}

func recoverFromGoroutine(fn func()) any {
	recovered := make(chan any, 1)
	go func() {
		defer func() { recovered <- recover() }()
		fn()
	}()
	return <-recovered
}

func runCircularGuarded(t *testing.T, fn func()) any {
	t.Helper()
	done := make(chan any, 1)
	go func() {
		defer func() { done <- recover() }()
		fn()
	}()
	select {
	case r := <-done:
		return r
	case <-time.After(circularDeadlockTimeout):
		t.Fatalf("resolution deadlocked (no bounded outcome within %s)", circularDeadlockTimeout)
		return nil
	}
}

func expectCircularPanic(t *testing.T, recovered any) {
	t.Helper()
	if err, ok := recovered.(error); !ok || !errors.Is(err, CircularInjectable) {
		t.Fatalf("expected CircularInjectable panic, got %v", recovered)
	}
}
