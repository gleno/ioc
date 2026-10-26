package ioc

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
)

type namedNumbers []int

func assertNotProvided[T any](t *testing.T, ctx context.Context) {
	t.Helper()
	t.Run("required", func(t *testing.T) {
		expectMissingPanic(t, "assignment-only match", func() { _ = GetProvided[T](ctx) })
	})
	t.Run("optional", func(t *testing.T) {
		defer func() {
			if recovered := recover(); recovered != nil {
				t.Fatalf("optional lookup panicked: %v", recovered)
			}
		}()
		value, found := GetOptionalProvided[T](ctx)
		var zero T
		if found || !reflect.DeepEqual(value, zero) {
			t.Fatalf("expected zero value and false, got %v, %v", value, found)
		}
	})
	t.Run("presence", func(t *testing.T) {
		defer func() {
			if recovered := recover(); recovered != nil {
				t.Fatalf("presence lookup panicked: %v", recovered)
			}
		}()
		if IsProvided[T](ctx) {
			t.Fatal("assignment-compatible but assertion-incompatible value must not match")
		}
	})
}

func TestConcreteResolutionUsesTypeAssertions(t *testing.T) {
	t.Run("named slice", func(t *testing.T) {
		assertNotProvided[[]int](t, WithProvided(context.Background(), namedNumbers{1}))
	})
	t.Run("receive-only channel", func(t *testing.T) {
		assertNotProvided[<-chan int](t, WithProvided(context.Background(), make(chan int)))
	})
	t.Run("send-only channel", func(t *testing.T) {
		assertNotProvided[chan<- int](t, WithProvided(context.Background(), make(chan int)))
	})
}

func TestAssignmentOnlyCandidatesDoNotCauseAmbiguity(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		name := "named first"
		values := []any{namedNumbers{1}, []int{2}}
		if reverse {
			name = "unnamed first"
			slices.Reverse(values)
		}
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("distinct concrete providers must not collide: %v", recovered)
				}
			}()
			ctx := WithProvided(context.Background(), values...)
			if got := GetProvided[[]int](ctx); !slices.Equal(got, []int{2}) {
				t.Fatalf("expected unnamed slice, got %v", got)
			}
			if got := GetProvided[namedNumbers](ctx); !slices.Equal(got, namedNumbers{1}) {
				t.Fatalf("expected named slice, got %v", got)
			}
		})
	}
}

func TestAssignmentOnlyOverrideDoesNotHideProvider(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("unrelated override must not hide the provider: %v", recovered)
		}
	}()
	ctx := WithProvided(context.Background(), []int{1})
	ctx = WithOverride(ctx, namedNumbers{2})
	if got := GetProvided[[]int](ctx); !slices.Equal(got, []int{1}) {
		t.Fatalf("expected original unnamed slice, got %v", got)
	}
	if got := GetProvided[namedNumbers](ctx); !slices.Equal(got, namedNumbers{2}) {
		t.Fatalf("expected named override, got %v", got)
	}
}

func TestAsUsesItsDeclaredParameterType(t *testing.T) {
	channel := make(chan int)
	ctx := WithProvided(context.Background(), As[[]int](namedNumbers{1}), As[<-chan int](channel))
	if got := GetProvided[[]int](ctx); !slices.Equal(got, []int{1}) {
		t.Fatalf("expected slice assigned into the pin type, got %v", got)
	}
	if got := GetProvided[<-chan int](ctx); got != channel {
		t.Fatal("expected channel assigned into the pin type")
	}
	if IsProvided[namedNumbers](ctx) || IsProvided[chan int](ctx) {
		t.Fatal("pins must match only their declared types")
	}
}

func TestAmbiguityCountsOnlyMatchesThroughNewestMatchingOverride(t *testing.T) {
	ctx := WithProvided(context.Background(), &emailService{})
	ctx = WithOverride(ctx, &emailService{})
	ctx = WithOverride(ctx, &_DoBServiceImpl{})
	ctx = WithProvided(ctx, &emailService{}, &smsService{})
	defer func() {
		recovered := recover()
		err, ok := recovered.(error)
		if !ok || !errors.Is(err, AmbiguousInjectable) {
			t.Fatalf("expected AmbiguousInjectable, got %v", recovered)
		}
		if !strings.Contains(err.Error(), "(3 matches)") {
			t.Fatalf("expected three matches through the matching override, got %v", err)
		}
	}()
	_ = GetProvided[Notifier](ctx)
}
