package ioc

import (
	"context"
	"reflect"
	"testing"
	"unsafe"
)

func TestNilUnsafePointerRejected(t *testing.T) {
	var pointer unsafe.Pointer
	for _, test := range []struct {
		name string
		run  func()
	}{
		{"WithProvided", func() { _ = WithProvided(context.Background(), pointer) }},
		{"WithOverride", func() { _ = WithOverride(context.Background(), pointer) }},
		{"As", func() { _ = As[unsafe.Pointer](pointer) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			expectInvalidPanic(t, test.name, test.run)
		})
	}
}

func TestAsRejectsEmptyInterfaceTarget(t *testing.T) {
	type Empty interface{}
	t.Run("any", func(t *testing.T) {
		expectInvalidPanic(t, "As[any]", func() { _ = As[any](42) })
	})
	t.Run("named empty interface", func(t *testing.T) {
		expectInvalidPanic(t, "As[Empty]", func() { _ = As[Empty](42) })
	})
}

func TestNamedEmptyInterfaceResolutionRejected(t *testing.T) {
	type Empty interface{}
	ctx := WithProvided(context.Background(), 42)

	expectInvalidPanic(t, "GetProvided[Empty]", func() { _ = GetProvided[Empty](ctx) })
	expectInvalidPanic(t, "GetOptionalProvided[Empty]", func() { _, _ = GetOptionalProvided[Empty](ctx) })
	expectInvalidPanic(t, "IsProvided[Empty]", func() { _ = IsProvided[Empty](ctx) })
}

func TestAsRejectsNilCollections(t *testing.T) {
	for _, test := range []struct {
		name string
		run  func()
	}{
		{"map", func() { _ = As[map[string]int](nil) }},
		{"slice", func() { _ = As[[]int](nil) }},
		{"channel", func() { _ = As[chan int](nil) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			expectInvalidPanic(t, "As nil "+test.name, test.run)
		})
	}
}

func TestValidValuesRemainInjectable(t *testing.T) {
	checkRegistrationRoundTrip(t, "unsafe pointer", unsafe.Pointer(new(int)))
	checkRegistrationRoundTrip(t, "zero number", 0)
	checkRegistrationRoundTrip(t, "false", false)
	checkRegistrationRoundTrip(t, "empty string", "")
	checkRegistrationRoundTrip(t, "empty struct", struct{}{})
	checkRegistrationRoundTrip(t, "empty map", map[string]int{})
	checkRegistrationRoundTrip(t, "empty slice", []int{})
	checkRegistrationRoundTrip(t, "channel", make(chan int))
}

func checkRegistrationRoundTrip[T any](t *testing.T, name string, value T) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		for _, test := range []struct {
			name     string
			register func(context.Context, ...any) context.Context
		}{
			{"WithProvided", WithProvided[context.Context]},
			{"WithOverride", WithOverride[context.Context]},
		} {
			t.Run(test.name, func(t *testing.T) {
				for _, provider := range []struct {
					name  string
					value any
				}{
					{"plain", value},
					{"pinned", As[T](value)},
				} {
					t.Run(provider.name, func(t *testing.T) {
						ctx := test.register(context.Background(), provider.value)
						got, found := GetOptionalProvided[T](ctx)
						if !found || !reflect.DeepEqual(got, value) {
							t.Fatalf("got (%#v, %v), want (%#v, true)", got, found, value)
						}
					})
				}
			})
		}
	})
}
