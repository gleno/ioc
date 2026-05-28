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

func expectMissingPanic(t *testing.T, what string, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("%s: expected MissingInjectable panic, did not panic", what)
		}
		if err, ok := r.(error); !ok || !errors.Is(err, MissingInjectable) {
			t.Fatalf("%s: expected MissingInjectable panic, got %v", what, r)
		}
	}()
	fn()
}

func expectPanicValue(t *testing.T, what string, want any, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("%s: expected panic %v, did not panic", what, want)
		}
		if r != want {
			t.Fatalf("%s: expected panic %v, got %v", what, want, r)
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

func TestFactoryPanicsPropagateByDefault(t *testing.T) {
	brokenCtx := func() context.Context {
		return WithProvided(context.Background(), func() ServeService {
			panic("factory boom")
		})
	}

	expectPanicValue(t, "IsProvided default panic", "factory boom", func() {
		_ = IsProvided[ServeService](brokenCtx())
	})
	expectPanicValue(t, "GetOptionalProvided default panic", "factory boom", func() {
		_, _ = GetOptionalProvided[ServeService](brokenCtx())
	})
	expectPanicValue(t, "GetProvided default panic", "factory boom", func() {
		_ = GetProvided[ServeService](brokenCtx())
	})
}

func TestSilencePanicsReportsAbsentOnBrokenMatchingFactory(t *testing.T) {
	brokenCtx := func() context.Context {
		ctx := SilencePanics(context.Background())
		return WithProvided(ctx, func() ServeService {
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

func TestSilencePanicsMakesGetProvidedPanicMissing(t *testing.T) {
	ctx := SilencePanics(context.Background())
	ctx = WithProvided(ctx, func() ServeService {
		panic("factory boom")
	})

	expectMissingPanic(t, "GetProvided with silenced factory panic", func() {
		_ = GetProvided[ServeService](ctx)
	})
}

func TestOnPanicFalsePropagatesOriginalPanic(t *testing.T) {
	var calls int
	ctx := OnPanic(context.Background(), func(recovered any) bool {
		calls++
		return false
	})
	ctx = WithProvided(ctx, func() ServeService {
		panic("factory boom")
	})

	expectPanicValue(t, "unhandled OnPanic", "factory boom", func() {
		_, _ = GetOptionalProvided[ServeService](ctx)
	})
	if calls != 1 {
		t.Fatalf("expected handler to be called once, got %d", calls)
	}
}

func TestOnPanicHandlersChainNewestToOldest(t *testing.T) {
	var order []string
	ctx := OnPanic(context.Background(), func(recovered any) bool {
		order = append(order, "old")
		return true
	})
	ctx = OnPanic(ctx, func(recovered any) bool {
		order = append(order, "new")
		return false
	})
	ctx = WithProvided(ctx, func() ServeService {
		panic("factory boom")
	})

	if value, found := GetOptionalProvided[ServeService](ctx); found || value != nil {
		t.Fatalf("expected handled factory panic to report absent, got (%v,%v)", value, found)
	}
	if len(order) != 2 || order[0] != "new" || order[1] != "old" {
		t.Fatalf("expected handlers to run newest-to-oldest, got %v", order)
	}
}

func TestOnPanicStopsAtFirstHandledPanic(t *testing.T) {
	var calls int
	ctx := OnPanic(context.Background(), func(recovered any) bool {
		calls++
		return false
	})
	ctx = OnPanic(ctx, func(recovered any) bool {
		calls++
		return true
	})
	ctx = WithProvided(ctx, func() ServeService {
		panic("factory boom")
	})

	_, found := GetOptionalProvided[ServeService](ctx)
	if found {
		t.Fatal("expected handled factory panic to report absent")
	}
	if calls != 1 {
		t.Fatalf("expected only the newest handler to be called, got %d calls", calls)
	}
}

func TestOnPanicInheritedThroughDerivedContext(t *testing.T) {
	parent := OnPanic(context.Background(), func(recovered any) bool {
		return true
	})
	child := context.WithValue(parent, "key", "value")
	child = WithProvided(child, func() ServeService {
		panic("factory boom")
	})

	if value, found := GetOptionalProvided[ServeService](child); found || value != nil {
		t.Fatalf("expected inherited panic handler to report absent, got (%v,%v)", value, found)
	}
}

func TestOnPanicHandlerPanicPropagates(t *testing.T) {
	ctx := OnPanic(context.Background(), func(recovered any) bool {
		panic("handler boom")
	})
	ctx = WithProvided(ctx, func() ServeService {
		panic("factory boom")
	})

	expectPanicValue(t, "panic handler panic", "handler boom", func() {
		_, _ = GetOptionalProvided[ServeService](ctx)
	})
}

func TestOnPanicRejectsNilHandler(t *testing.T) {
	expectInvalidPanic(t, "nil panic handler", func() {
		_ = OnPanic(context.Background(), nil)
	})
}

func TestOnPanicDoesNotHandleNonFactoryPanics(t *testing.T) {
	var calls int
	ctx := OnPanic(context.Background(), func(recovered any) bool {
		calls++
		return true
	})

	func() {
		defer expectAmbiguousPanic(t, "ambiguous provider")
		ambiguous := WithProvided(ctx, &_DoAServiceImpl{}, &_DoAServiceImpl2{})
		_ = GetProvided[ServiceA](ambiguous)
	}()
	if calls != 0 {
		t.Fatalf("ambiguous provider called panic handler %d times", calls)
	}

	expectMissingPanic(t, "missing provider", func() {
		_ = GetProvided[ServiceA](ctx)
	})
	if calls != 0 {
		t.Fatalf("missing provider called panic handler %d times", calls)
	}

	expectInvalidPanic(t, "resolve any", func() {
		_ = GetProvided[any](ctx)
	})
	if calls != 0 {
		t.Fatalf("resolve any called panic handler %d times", calls)
	}

	expectInvalidPanic(t, "invalid registration", func() {
		_ = WithProvided(ctx, nil)
	})
	if calls != 0 {
		t.Fatalf("invalid registration called panic handler %d times", calls)
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
