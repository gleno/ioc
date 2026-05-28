package ioc

import (
	"context"
	"errors"
	"testing"
)

type Notifier interface {
	Send() string
}

type Emailer interface {
	Send() string
	Compose() string
}

type SMSSender interface {
	Send() string
	SetNumber() string
}

type emailService struct{}

func (*emailService) Send() string    { return "email" }
func (*emailService) Compose() string { return "compose" }

type smsService struct{}

func (*smsService) Send() string      { return "sms" }
func (*smsService) SetNumber() string { return "number" }

func expectAmbiguousPanic(t *testing.T, what string) {
	t.Helper()
	r := recover()
	if r == nil {
		t.Fatalf("%s: expected AmbiguousInjectable panic, did not panic", what)
	}
	if err, ok := r.(error); !ok || !errors.Is(err, AmbiguousInjectable) {
		t.Fatalf("%s: expected AmbiguousInjectable panic, got %v", what, r)
	}
}

func TestPlainCollisionPanics(t *testing.T) {
	defer expectAmbiguousPanic(t, "plain collision")
	ctx := WithProvided(context.Background(), &emailService{}, &smsService{})
	_ = GetProvided[Notifier](ctx)
}

func TestGetOptionalProvidedPanicsOnAmbiguity(t *testing.T) {
	defer expectAmbiguousPanic(t, "GetOptionalProvided ambiguity")
	ctx := WithProvided(context.Background(), &_DoAServiceImpl{}, &_DoAServiceImpl2{})
	_, _ = GetOptionalProvided[ServiceA](ctx)
}

func TestIsProvidedPanicsOnAmbiguity(t *testing.T) {
	defer expectAmbiguousPanic(t, "IsProvided ambiguity")
	ctx := WithProvided(context.Background(), &_DoAServiceImpl{}, &_DoAServiceImpl2{})
	_ = IsProvided[ServiceA](ctx)
}

func TestOverrideShadowsOnlyOverlappingTypes(t *testing.T) {
	ctx := WithProvided(context.Background(), &_DoAServiceImpl{})
	ctx = WithOverride(ctx, &_DoBServiceImpl{})

	if GetProvided[ServiceA](ctx).DoA() != "A" {
		t.Fatal("an override of an unrelated type must not shadow ServiceA")
	}
	if GetProvided[ServiceB](ctx).DoB() != "B" {
		t.Fatal("expected the override's ServiceB")
	}
}

func TestNewerNormalProvideOnOverridePanics(t *testing.T) {
	defer expectAmbiguousPanic(t, "normal provide atop override")
	ctx := WithOverride(context.Background(), &_DoAServiceImpl{})
	ctx = WithProvided(ctx, &_DoAServiceImpl2{})
	_ = GetProvided[ServiceA](ctx)
}

func TestAsPinsToExactType(t *testing.T) {
	ctx := WithProvided(context.Background(), As[Emailer](&emailService{}))

	if GetProvided[Emailer](ctx).Send() != "email" {
		t.Fatal("As[Emailer] must resolve as Emailer")
	}
	if IsProvided[Notifier](ctx) {
		t.Fatal("As[Emailer] must not resolve as the Notifier sub-interface it incidentally satisfies")
	}
	if IsProvided[SMSSender](ctx) {
		t.Fatal("As[Emailer] must not resolve as an unrelated interface")
	}
}

func TestAsDissolvesIncidentalCollision(t *testing.T) {
	ctx := WithProvided(context.Background(),
		As[Emailer](&emailService{}),
		As[SMSSender](&smsService{}),
	)

	if _, found := GetOptionalProvided[Notifier](ctx); found {
		t.Fatal("once both impls are pinned to their real roles, the incidental Notifier match must dissolve")
	}
	if GetProvided[Emailer](ctx).Compose() != "compose" {
		t.Fatal("expected the pinned Emailer")
	}
	if GetProvided[SMSSender](ctx).SetNumber() != "number" {
		t.Fatal("expected the pinned SMSSender")
	}
}

func TestAsRejectsNilImpl(t *testing.T) {
	expectInvalidPanic(t, "As nil interface", func() {
		_ = As[ServeService](nil)
	})
	expectInvalidPanic(t, "As typed-nil pointer", func() {
		var nilImpl *_ServeServiceImpl = nil
		_ = As[ServeService](nilImpl)
	})
}
