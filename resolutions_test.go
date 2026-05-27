package ioc

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type _NilMap map[string]int
type _NilSlice []int
type _NilChan chan int

func expectInvalidPanic(t *testing.T, what string, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("%s: expected InvalidInjectable panic, did not panic", what)
		}
		if err, ok := r.(error); !ok || !errors.Is(err, InvalidInjectable) {
			t.Fatalf("%s: expected InvalidInjectable panic, got %v", what, r)
		}
	}()
	fn()
}

func TestSelfDependencyPanicsCircular(t *testing.T) {
	r := runGuarded(t, func() {
		var ctx context.Context
		ctx = WithProvided(context.Background(), func() *_ServeServiceImpl {
			_ = GetProvided[ServeService](ctx)
			return &_ServeServiceImpl{}
		})
		_ = GetProvided[ServeService](ctx)
	})
	if err, ok := r.(error); !ok || !errors.Is(err, CircularInjectable) {
		t.Fatalf("expected CircularInjectable panic, got %v", r)
	}
}

func TestMutualCyclePanicsCircular(t *testing.T) {
	r := runGuarded(t, func() {
		var ctx context.Context
		ctx = WithProvided(context.Background(),
			func() *_ServeServiceImpl { _ = GetProvided[DoSomethingService](ctx); return &_ServeServiceImpl{} },
			func() *_DoSomethingServiceImpl { _ = GetProvided[ServeService](ctx); return &_DoSomethingServiceImpl{} },
		)
		_ = GetProvided[ServeService](ctx)
	})
	if err, ok := r.(error); !ok || !errors.Is(err, CircularInjectable) {
		t.Fatalf("expected CircularInjectable panic, got %v", r)
	}
}

func runGuarded(t *testing.T, fn func()) any {
	t.Helper()
	done := make(chan any, 1)
	go func() {
		defer func() { done <- recover() }()
		fn()
		done <- nil
	}()
	select {
	case r := <-done:
		return r
	case <-time.After(2 * time.Second):
		t.Fatal("resolution deadlocked (no bounded outcome within 2s)")
		return nil
	}
}

func TestFactoryResolvingDistinctDependencyIsNotCircular(t *testing.T) {
	ctx := WithProvided(context.Background(), &_DoSomethingServiceImpl{})
	ctx = WithProvided(ctx, func() *_ServeServiceImpl {
		_ = GetProvided[DoSomethingService](ctx)
		return &_ServeServiceImpl{}
	})
	if GetProvided[ServeService](ctx) == nil {
		t.Fatal("a factory resolving a distinct dependency must resolve, not be flagged circular")
	}
}

func TestResolveAnyIsRejected(t *testing.T) {
	var calls int
	ctx := WithProvided(context.Background(), &_ServeServiceImpl{})
	ctx = WithProvided(ctx, func() *_DoSomethingServiceImpl { calls++; return &_DoSomethingServiceImpl{} })

	expectInvalidPanic(t, "GetProvided[any]", func() { _ = GetProvided[any](ctx) })
	expectInvalidPanic(t, "GetOptionalProvided[any]", func() { _, _ = GetOptionalProvided[any](ctx) })
	expectInvalidPanic(t, "IsProvided[any]", func() { _ = IsProvided[any](ctx) })

	if calls != 0 {
		t.Fatalf("resolving any invoked a factory %d times; must be rejected before any resolution", calls)
	}
}

func TestQueryAPIsReportAbsentOnBrokenMatchingFactory(t *testing.T) {
	brokenCtx := func() context.Context {
		return WithProvided(context.Background(), func() ServeService {
			var nilImpl *_ServeServiceImpl = nil
			return nilImpl
		})
	}

	if IsProvided[ServeService](brokenCtx()) {
		t.Fatal("IsProvided must report false for a broken matching factory")
	}
	if v, found := GetOptionalProvided[ServeService](brokenCtx()); found || v != nil {
		t.Fatalf("GetOptionalProvided must report (nil,false) for a broken matching factory; got (%v,%v)", v, found)
	}
}

func TestGetProvidedStillPanicsOnBrokenMatchingFactory(t *testing.T) {
	ctx := WithProvided(context.Background(), func() ServeService {
		var nilImpl *_ServeServiceImpl = nil
		return nilImpl
	})
	expectInvalidPanic(t, "GetProvided broken factory", func() { _ = GetProvided[ServeService](ctx) })
}

func TestFactoryReturningNilCollectionsRejected(t *testing.T) {
	expectInvalidPanic(t, "factory nil map", func() {
		_ = GetProvided[_NilMap](WithProvided(context.Background(), func() _NilMap { return nil }))
	})
	expectInvalidPanic(t, "factory nil slice", func() {
		_ = GetProvided[_NilSlice](WithProvided(context.Background(), func() _NilSlice { return nil }))
	})
	expectInvalidPanic(t, "factory nil chan", func() {
		_ = GetProvided[_NilChan](WithProvided(context.Background(), func() _NilChan { return nil }))
	})
}

func TestDirectNilCollectionsRejected(t *testing.T) {
	expectInvalidPanic(t, "direct nil map", func() {
		var m _NilMap
		_ = WithProvided(context.Background(), m)
	})
	expectInvalidPanic(t, "direct nil slice", func() {
		var s _NilSlice
		_ = WithProvided(context.Background(), s)
	})
	expectInvalidPanic(t, "direct nil chan", func() {
		var c _NilChan
		_ = WithProvided(context.Background(), c)
	})
}

func TestConcurrentResolutionDoesNotFalselyDetectCircular(t *testing.T) {
	var calls atomic.Int64
	ctx := WithProvided(context.Background(), func() *_ServeServiceImpl {
		calls.Add(1)
		return &_ServeServiceImpl{}
	})

	var wg sync.WaitGroup
	for range 64 {
		wg.Go(func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("concurrent resolution panicked: %v", r)
				}
			}()
			_ = GetProvided[ServeService](ctx)
		})
	}
	wg.Wait()

	if calls.Load() != 1 {
		t.Fatalf("factory invoked %d times under concurrency; OnceValue must memoize to 1", calls.Load())
	}
}
