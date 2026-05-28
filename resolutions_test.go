package ioc

import (
	"context"
	"errors"
	"testing"
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

func TestResolveAnyIsRejected(t *testing.T) {
	ctx := WithProvided(context.Background(), &_ServeServiceImpl{})

	expectInvalidPanic(t, "GetProvided[any]", func() { _ = GetProvided[any](ctx) })
	expectInvalidPanic(t, "GetOptionalProvided[any]", func() { _, _ = GetOptionalProvided[any](ctx) })
	expectInvalidPanic(t, "IsProvided[any]", func() { _ = IsProvided[any](ctx) })
}

func TestGetProvidedNoMatchingInterface(t *testing.T) {
	ctx := WithProvided(context.Background(), &_ServeServiceImpl{})

	expectMissingPanic(t, "no matching injectable", func() {
		_ = GetProvided[DoSomethingService](ctx)
	})
}

func TestIsProvidedReturnsTrue(t *testing.T) {
	ctx := WithProvided(context.Background(), &_ServeServiceImpl{})

	if !IsProvided[ServeService](ctx) {
		t.Fatal("expected IsProvided to return true")
	}
}

func TestIsProvidedReturnsFalse(t *testing.T) {
	if IsProvided[ServeService](context.Background()) {
		t.Fatal("expected IsProvided to return false")
	}
}

func TestGetOptionalProvidedReturnsValue(t *testing.T) {
	ctx := WithProvided(context.Background(), &_ServeServiceImpl{})

	value, found := GetOptionalProvided[ServeService](ctx)
	if !found {
		t.Fatal("expected found to be true")
	}
	if value.Serve() != "Service is serving" {
		t.Fatal("expected service to work correctly")
	}
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

func TestWithProvidedRejectsTypedNilPointer(t *testing.T) {
	expectInvalidPanic(t, "typed-nil pointer", func() {
		var nilImpl *_ServeServiceImpl
		_ = WithProvided(context.Background(), nilImpl)
	})
}
