package ioc

import (
	"context"
	"errors"
	"testing"
)

type ServeService interface {
	Serve() string
}

type _ServeServiceImpl struct{}

func (s *_ServeServiceImpl) Serve() string {
	return "Service is serving"
}

type DoSomethingService interface {
	DoSomething() string
}

type _DoSomethingServiceImpl struct{}

func (a *_DoSomethingServiceImpl) DoSomething() string {
	return "Another service is doing something"
}

type AdvancedService interface {
	ServeService
	AdvancedServe() string
}

type AdvancedServiceImpl struct{}

func (a *AdvancedServiceImpl) Serve() string {
	return "Advanced Service is serving"
}

func (a *AdvancedServiceImpl) AdvancedServe() string {
	return "Advanced Service is serving advanced"
}

type _serviceContext struct {
	context.Context
}

type ServiceA interface {
	DoA() string
}

type ServiceB interface {
	DoB() string
}

type _DoAServiceImpl struct{}

func (s *_DoAServiceImpl) DoA() string {
	return "A"
}

type _DoBServiceImpl struct{}

func (s *_DoBServiceImpl) DoB() string {
	return "B"
}

type _DoAServiceImpl2 struct{}

func (s *_DoAServiceImpl2) DoA() string {
	return "A-Alternate"
}

func TestWithProvidedAndGetProvided(t *testing.T) {
	service := &_ServeServiceImpl{}
	ctx := WithProvided(context.Background(), service)

	retrievedService := GetProvided[ServeService](ctx)
	if retrievedService == nil {
		t.Fatal("expected to retrieve ServeService, got nil")
	}
	if retrievedService.Serve() != service.Serve() {
		t.Fatal("retrieved service does not match the provided service")
	}
}

func TestGetProvidedMissingInjectable(t *testing.T) {
	defer func() {
		r := recover()
		if err, ok := r.(error); !ok || !errors.Is(err, MissingInjectable) {
			t.Fatalf("expected MissingInjectable panic, got %v", r)
		}
	}()

	_ = GetProvided[ServeService](context.Background())
}

func TestWithProvidedMultipleInjectablesAtOnce(t *testing.T) {
	ctx := WithProvided(context.Background(), &_ServeServiceImpl{}, &_DoSomethingServiceImpl{})

	if GetProvided[ServeService](ctx).Serve() != "Service is serving" {
		t.Fatal("expected ServeService to resolve")
	}
	if GetProvided[DoSomethingService](ctx).DoSomething() != "Another service is doing something" {
		t.Fatal("expected DoSomethingService to resolve")
	}
}

func TestGetProvidedWithNestedContexts(t *testing.T) {
	parent := WithProvided(context.Background(), &_ServeServiceImpl{})
	child := context.WithValue(parent, "someKey", "someValue")

	if GetProvided[ServeService](child).Serve() != "Service is serving" {
		t.Fatal("expected child context to inherit provided service")
	}
}

func TestRetrieveSpecificInterface(t *testing.T) {
	ctx := WithProvided(context.Background(), &AdvancedServiceImpl{})

	if GetProvided[AdvancedService](ctx).AdvancedServe() != "Advanced Service is serving advanced" {
		t.Fatal("expected AdvancedService to resolve")
	}
	if GetProvided[ServeService](ctx).Serve() != "Advanced Service is serving" {
		t.Fatal("expected ServeService to resolve from advanced implementation")
	}
}

func TestAmbiguityDueToOverlappingInterfaces(t *testing.T) {
	defer func() {
		r := recover()
		if err, ok := r.(error); !ok || !errors.Is(err, AmbiguousInjectable) {
			t.Fatalf("expected AmbiguousInjectable panic, got %v", r)
		}
	}()

	ctx := WithProvided(context.Background(), &_ServeServiceImpl{}, &AdvancedServiceImpl{})
	_ = GetProvided[ServeService](ctx)
}

func TestOverrideResolvesOverlappingInterfaces(t *testing.T) {
	ctx := WithProvided(context.Background(), &_ServeServiceImpl{})
	ctx = WithOverride(ctx, &AdvancedServiceImpl{})

	if GetProvided[ServeService](ctx).Serve() != "Advanced Service is serving" {
		t.Fatal("expected override to resolve advanced implementation")
	}
}

func TestNestedOverride(t *testing.T) {
	ctx := WithProvided(context.Background(), &_DoAServiceImpl{})
	ctx = WithProvided(ctx, &_DoBServiceImpl{})
	ctx = WithOverride(ctx, &_DoAServiceImpl2{})

	if GetProvided[ServiceA](ctx).DoA() != "A-Alternate" {
		t.Fatal("expected ServiceA override")
	}
	if GetProvided[ServiceB](ctx).DoB() != "B" {
		t.Fatal("expected unrelated ServiceB to remain available")
	}
}

func TestGetProvidedFromDeepStoredContext(t *testing.T) {
	customCtx := &_serviceContext{Context: context.Background()}
	ctx := WithProvided(customCtx, &_DoSomethingServiceImpl{})
	ctx = WithProvided(ctx, &_DoBServiceImpl{})

	if GetProvided[DoSomethingService](ctx).DoSomething() != "Another service is doing something" {
		t.Fatal("expected to find service from deeply stored context")
	}
}

func TestForeignStringKeyDoesNotCorruptResolution(t *testing.T) {
	ctx := WithProvided(context.Background(), &_ServeServiceImpl{})
	ctx = context.WithValue(ctx, "ioc", "some foreign value")

	if !IsProvided[ServeService](ctx) {
		t.Fatal("foreign string key must not shadow the ioc chain")
	}
}

func TestGetConcreteValue(t *testing.T) {
	ctx := WithProvided(context.Background(), &_DoAServiceImpl{})

	value, found := GetOptionalProvided[*_DoAServiceImpl](ctx)
	if !found {
		t.Fatal("expected to find concrete type")
	}
	if value.DoA() != "A" {
		t.Fatal("expected concrete value to work")
	}
}

func TestGetOptionalProvidedReturnsNotFound(t *testing.T) {
	value, found := GetOptionalProvided[ServeService](context.Background())
	if found {
		t.Fatal("expected found to be false")
	}
	if value != nil {
		t.Fatal("expected zero value")
	}
}

func TestWithProvidedRejectsFunction(t *testing.T) {
	expectInvalidPanic(t, "WithProvided function", func() {
		_ = WithProvided(context.Background(), func() *_ServeServiceImpl {
			return &_ServeServiceImpl{}
		})
	})
}

func TestWithOverrideRejectsFunction(t *testing.T) {
	expectInvalidPanic(t, "WithOverride function", func() {
		_ = WithOverride(context.Background(), func() *_ServeServiceImpl {
			return &_ServeServiceImpl{}
		})
	})
}

func TestWithProvidedRejectsTypedNilFunction(t *testing.T) {
	expectInvalidPanic(t, "WithProvided typed-nil function", func() {
		var nilFunc func() *_ServeServiceImpl
		_ = WithProvided(context.Background(), nilFunc)
	})
}

func TestAsRejectsFunction(t *testing.T) {
	type Handler func() string
	expectInvalidPanic(t, "As function", func() {
		_ = As[Handler](func() string { return "handled" })
	})
}

func TestAsRejectsTypedNilFunction(t *testing.T) {
	type Handler func() string
	expectInvalidPanic(t, "As typed-nil function", func() {
		var nilHandler Handler
		_ = As[Handler](nilHandler)
	})
}
