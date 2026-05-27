package ioc

import (
	"context"
	"errors"
	"testing"
)

// Mock interfaces and structs for testing
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

func TestWithProvidedAndGetProvided(t *testing.T) {
	ctx := context.Background()
	service := &_ServeServiceImpl{}

	// Provide the service implementation
	ctx = WithProvided(ctx, service)

	// Retrieve the service as the Service interface
	retrievedService := GetProvided[ServeService](ctx)
	if retrievedService == nil {
		t.Fatal("Expected to retrieve Service, got nil")
	}

	// Verify that the retrieved service is the same as the provided one
	if retrievedService.Serve() != service.Serve() {
		t.Fatal("Retrieved service does not match the provided service")
	}
}

func TestGetProvidedMissingInjectable(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok && errors.Is(err, MissingInjectable) {
				// Expected behavior
			} else {
				t.Fatalf("Expected MissingInjectable panic, got %v", r)
			}
		} else {
			t.Fatal("Expected panic for missing injectable, but code did not panic")
		}
	}()

	var ctx = context.Background()
	// Attempt to retrieve an injectable without providing it first
	_ = GetProvided[ServeService](ctx)
}

func TestWithProvidedMultipleInjectables(t *testing.T) {
	var ctx = context.Background()
	service := &_ServeServiceImpl{}
	anotherService := &_DoSomethingServiceImpl{}

	// Provide multiple injectables
	ctx = WithProvided(ctx, service)
	ctx = WithProvided(ctx, anotherService)

	// Retrieve the first injectable
	retrievedService := GetProvided[ServeService](ctx)
	if retrievedService == nil {
		t.Fatal("Expected to retrieve Service, got nil")
	}

	// Retrieve the second injectable
	retrievedAnotherService := GetProvided[DoSomethingService](ctx)
	if retrievedAnotherService == nil {
		t.Fatal("Expected to retrieve AnotherService, got nil")
	}

	// Verify the functionalities
	if retrievedService.Serve() != "Service is serving" {
		t.Fatal("Retrieved service does not perform as expected")
	}

	if retrievedAnotherService.DoSomething() != "Another service is doing something" {
		t.Fatal("Retrieved another service does not perform as expected")
	}
}

func TestGetProvidedWithNestedContexts(t *testing.T) {
	ctx := context.Background()
	service := &_ServeServiceImpl{}

	// Provide the service in the parent context
	parentCtx := WithProvided(ctx, service)
	// Create a child context without providing the service
	childCtx := context.WithValue(parentCtx, "someKey", "someValue")

	// Retrieve the service from the child context
	retrievedService := GetProvided[ServeService](childCtx)
	if retrievedService == nil {
		t.Fatal("Expected to retrieve Service from child context, got nil")
	}

	// Verify that the retrieved service is the same as the provided one
	if retrievedService.Serve() != service.Serve() {
		t.Fatal("Retrieved service from child context does not match the provided service")
	}
}

func TestWithProvidedAndGetProvided1(t *testing.T) {
	ctx := context.Background()
	service := &_ServeServiceImpl{}

	// Provide the service implementation
	ctx = WithProvided(ctx, service)

	// Retrieve the service as the Service interface
	retrievedService := GetProvided[ServeService](ctx)
	if retrievedService == nil {
		t.Fatal("Expected to retrieve Service, got nil")
	}

	// Verify that the retrieved service is the same as the provided one
	if retrievedService.Serve() != service.Serve() {
		t.Fatal("Retrieved service does not match the provided service")
	}
}

func TestWithProvidedMultipleInjectablesAtOnce(t *testing.T) {
	ctx := context.Background()
	service := &_ServeServiceImpl{}
	anotherService := &_DoSomethingServiceImpl{}

	// Provide multiple injectables at once
	ctx = WithProvided(ctx, service, anotherService)

	// Retrieve the Service interface
	retrievedService := GetProvided[ServeService](ctx)
	if retrievedService == nil {
		t.Fatal("Expected to retrieve Service, got nil")
	}

	// Retrieve the AnotherService interface
	retrievedAnotherService := GetProvided[DoSomethingService](ctx)
	if retrievedAnotherService == nil {
		t.Fatal("Expected to retrieve AnotherService, got nil")
	}

	// Verify the functionalities
	if retrievedService.Serve() != "Service is serving" {
		t.Fatal("Retrieved service does not perform as expected")
	}

	if retrievedAnotherService.DoSomething() != "Another service is doing something" {
		t.Fatal("Retrieved another service does not perform as expected")
	}
}

func TestGetProvidedMissingInjectable1(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok && errors.Is(err, MissingInjectable) {
				// Expected behavior
			} else {
				t.Fatalf("Expected MissingInjectable panic, got %v", r)
			}
		} else {
			t.Fatal("Expected panic for missing injectable, but code did not panic")
		}
	}()

	ctx := context.Background()
	// Attempt to retrieve an injectable without providing it first
	_ = GetProvided[ServeService](ctx)
}

func TestWithProvidedMultipleInjectablesSeparateCalls(t *testing.T) {
	ctx := context.Background()
	service := &_ServeServiceImpl{}
	anotherService := &_DoSomethingServiceImpl{}

	// Provide multiple injectables in separate calls
	ctx = WithProvided(ctx, service)
	ctx = WithProvided(ctx, anotherService)

	// Retrieve the Service interface
	retrievedService := GetProvided[ServeService](ctx)
	if retrievedService == nil {
		t.Fatal("Expected to retrieve Service, got nil")
	}

	// Retrieve the AnotherService interface
	retrievedAnotherService := GetProvided[DoSomethingService](ctx)
	if retrievedAnotherService == nil {
		t.Fatal("Expected to retrieve AnotherService, got nil")
	}

	// Verify the functionalities
	if retrievedService.Serve() != "Service is serving" {
		t.Fatal("Retrieved service does not perform as expected")
	}

	if retrievedAnotherService.DoSomething() != "Another service is doing something" {
		t.Fatal("Retrieved another service does not perform as expected")
	}
}

func TestGetProvidedWithNestedContexts1(t *testing.T) {
	ctx := context.Background()
	service := &_ServeServiceImpl{}

	// Provide the service in the parent context
	parentCtx := WithProvided(ctx, service)
	// Create a child context without providing the service
	childCtx := context.WithValue(parentCtx, "someKey", "someValue")

	// Retrieve the service from the child context
	retrievedService := GetProvided[ServeService](childCtx)
	if retrievedService == nil {
		t.Fatal("Expected to retrieve Service from child context, got nil")
	}

	// Verify that the retrieved service is the same as the provided one
	if retrievedService.Serve() != service.Serve() {
		t.Fatal("Retrieved service from child context does not match the provided service")
	}
}

func TestAmbiguousInjectablePanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok && errors.Is(err, AmbiguousInjectable) {
				return
			}
			t.Fatalf("Expected AmbiguousInjectable panic, got %v", r)
		}
		t.Fatal("Expected panic for two providers matching ServeService, but code did not panic")
	}()

	ctx := context.Background()
	ctx = WithProvided(ctx, &AdvancedServiceImpl{}, &_ServeServiceImpl{})

	_ = GetProvided[ServeService](ctx)
}

func TestRetrieveSpecificInterface(t *testing.T) {
	ctx := context.Background()
	advancedService := &AdvancedServiceImpl{}

	// Provide only AdvancedServiceImpl
	ctx = WithProvided(ctx, advancedService)

	// Retrieve the AdvancedService interface
	retrievedAdvancedService := GetProvided[AdvancedService](ctx)
	if retrievedAdvancedService == nil {
		t.Fatal("Expected to retrieve AdvancedService, got nil")
	}

	// Verify that the retrieved service performs correctly
	if retrievedAdvancedService.AdvancedServe() != "Advanced Service is serving advanced" {
		t.Fatal("Retrieved advanced service does not perform as expected")
	}

	// Retrieve the Service interface
	retrievedService := GetProvided[ServeService](ctx)
	if retrievedService == nil {
		t.Fatal("Expected to retrieve Service, got nil")
	}

	// Verify that the retrieved service performs correctly
	if retrievedService.Serve() != "Advanced Service is serving" {
		t.Fatal("Retrieved service does not perform as expected")
	}
}

func TestNoAmbiguityWithSingleImplementation(t *testing.T) {
	ctx := context.Background()
	advancedService := &AdvancedServiceImpl{}

	// Provide only AdvancedServiceImpl
	ctx = WithProvided(ctx, advancedService)

	// Retrieve the Service interface
	retrievedService := GetProvided[ServeService](ctx)
	if retrievedService == nil {
		t.Fatal("Expected to retrieve Service, got nil")
	}

	// Verify that the retrieved service is the advanced service
	if retrievedService.Serve() != "Advanced Service is serving" {
		t.Fatal("Retrieved service does not perform as expected")
	}
}

func TestAmbiguityDueToOverlappingInterfaces(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok && errors.Is(err, AmbiguousInjectable) {
				return
			}
			t.Fatalf("Expected AmbiguousInjectable panic, got %v", r)
		}
		t.Fatal("Expected panic for overlapping ServeService providers, but code did not panic")
	}()

	ctx := context.Background()
	ctx = WithProvided(ctx, &_ServeServiceImpl{}, &AdvancedServiceImpl{})

	_ = GetProvided[ServeService](ctx)
}

func TestOverrideResolvesOverlappingInterfaces(t *testing.T) {
	ctx := context.Background()
	ctx = WithProvided(ctx, &_ServeServiceImpl{})
	ctx = WithOverride(ctx, &AdvancedServiceImpl{})

	retrievedService := GetProvided[ServeService](ctx)
	if retrievedService.Serve() != "Advanced Service is serving" {
		t.Fatalf("Expected AdvancedServiceImpl via override, got: %s", retrievedService.Serve())
	}
}

type AdvancedServiceImpl struct{}

func (a *AdvancedServiceImpl) Serve() string {
	return "Advanced Service is serving"
}

func (a *AdvancedServiceImpl) AdvancedServe() string {
	return "Advanced Service is serving advanced"
}

type AdvancedService interface {
	ServeService
	AdvancedServe() string
}

// Tests for callable injectables

func TestWithProvidedCallableInjectable(t *testing.T) {
	var ctx = context.Background()

	// Create a factory function that returns a service implementation
	var factory = func() *_ServeServiceImpl {
		return &_ServeServiceImpl{}
	}

	ctx = WithProvided(ctx, factory)

	var retrievedService = GetProvided[ServeService](ctx)
	if retrievedService == nil {
		t.Fatal("Expected to retrieve Service from callable injectable, got nil")
	}

	if retrievedService.Serve() != "Service is serving" {
		t.Fatal("Retrieved service from callable injectable does not perform as expected")
	}
}

func TestWithProvidedCallableInjectableReturnsNil(t *testing.T) {
	ctx := context.Background()

	// Create a factory function that returns nil
	factory := func() *_ServeServiceImpl {
		return nil
	}

	// WithProvided should not panic - the function is stored as lazy factory
	ctx = WithProvided(ctx, factory)

	// The panic should happen when we try to GetProvided (when the factory is invoked)
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok && errors.Is(err, InvalidInjectable) {
				// Expected behavior - factory returning nil should panic on first use
			} else {
				t.Fatalf("Expected InvalidInjectable panic, got %v", r)
			}
		} else {
			t.Fatal("Expected panic for callable injectable returning nil, but code did not panic")
		}
	}()

	// This should panic because the factory returns nil when called
	_ = GetProvided[ServeService](ctx)
}

func TestWithProvidedCallableInjectableWithParameters(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok && errors.Is(err, InvalidInjectable) {
				// Expected behavior - functions with parameters should be rejected
			} else {
				t.Fatalf("Expected InvalidInjectable panic, got %v", r)
			}
		} else {
			t.Fatal("Expected panic for callable injectable with parameters, but code did not panic")
		}
	}()

	ctx := context.Background()

	// Create a factory function that takes parameters
	factoryWithParams := func(name string) *_ServeServiceImpl {
		return &_ServeServiceImpl{}
	}

	// This should panic because we can't call a function with parameters
	_ = WithProvided(ctx, factoryWithParams)
}

func TestWithProvidedCallableInjectableMixedWithRegular(t *testing.T) {
	ctx := context.Background()

	// Create a regular service
	regularService := &_ServeServiceImpl{}

	// Create a factory for another service
	anotherFactory := func() *_DoSomethingServiceImpl {
		return &_DoSomethingServiceImpl{}
	}

	// Provide both regular and callable injectables
	ctx = WithProvided(ctx, regularService, anotherFactory)

	// Retrieve both services
	retrievedService := GetProvided[ServeService](ctx)
	if retrievedService == nil {
		t.Fatal("Expected to retrieve Service, got nil")
	}

	retrievedAnotherService := GetProvided[DoSomethingService](ctx)
	if retrievedAnotherService == nil {
		t.Fatal("Expected to retrieve AnotherService, got nil")
	}

	// Verify both work correctly
	if retrievedService.Serve() != "Service is serving" {
		t.Fatal("Retrieved regular service does not perform as expected")
	}

	if retrievedAnotherService.DoSomething() != "Another service is doing something" {
		t.Fatal("Retrieved service from callable injectable does not perform as expected")
	}
}

func TestWithProvidedCallableInjectableReturningInterface(t *testing.T) {
	ctx := context.Background()

	// Create a factory that returns an interface directly
	factory := func() ServeService {
		return &_ServeServiceImpl{}
	}

	// Provide the factory
	ctx = WithProvided(ctx, factory)

	// Retrieve the service
	retrievedService := GetProvided[ServeService](ctx)
	if retrievedService == nil {
		t.Fatal("Expected to retrieve Service from interface-returning callable injectable, got nil")
	}

	// Verify functionality
	if retrievedService.Serve() != "Service is serving" {
		t.Fatal("Retrieved service from interface-returning callable injectable does not perform as expected")
	}
}

func TestWithProvidedVoidFunction(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok && errors.Is(err, InvalidInjectable) {
				// Expected behavior - void functions should be rejected
			} else {
				t.Fatalf("Expected InvalidInjectable panic, got %v", r)
			}
		} else {
			t.Fatal("Expected panic for void function callable injectable, but code did not panic")
		}
	}()

	ctx := context.Background()

	// Create a function variable that doesn't return anything (void function)
	voidFunc := func() {
		// Do nothing
	}

	// This should panic because void functions cannot be injectables
	_ = WithProvided(ctx, voidFunc)
}

func TestWithProvidedTypedNilFunction(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok && errors.Is(err, InvalidInjectable) {
				// Expected behavior - a typed nil func value is an invalid injectable
			} else {
				t.Fatalf("Expected InvalidInjectable panic, got %v", r)
			}
		} else {
			t.Fatal("Expected panic for typed nil function injectable, but code did not panic")
		}
	}()

	ctx := context.Background()

	var nilFunc func() *_ServeServiceImpl = nil

	// A typed nil func slips past the injectable == nil guard; it must still be rejected.
	_ = WithProvided(ctx, nilFunc)
}

func TestWithProvidedNilDirectly(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok && errors.Is(err, InvalidInjectable) {
				// Expected behavior - nil injectables should be rejected
			} else {
				t.Fatalf("Expected InvalidInjectable panic, got %v", r)
			}
		} else {
			t.Fatal("Expected panic for nil injectable, but code did not panic")
		}
	}()

	ctx := context.Background()

	// Pass nil directly as injectable
	_ = WithProvided(ctx, nil)
}

func TestWithProvidedCallableReturningNonPointer(t *testing.T) {
	var ctx = context.Background()

	var factory = func() string {
		return "not a pointer"
	}

	ctx = WithProvided(ctx, factory)

	// The panic should happen when we try to GetProvided (when the factory is invoked)
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok && errors.Is(err, MissingInjectable) {
				// Expected behavior - functions returning non-pointers should be rejected on first use
			} else {
				t.Fatalf("Expected InvalidInjectable panic, got %v", r)
			}
		} else {
			t.Fatal("Expected panic for function returning non-pointer, but code did not panic")
		}
	}()

	// This should panic because the factory returns a non-pointer when called
	var item = GetProvided[ServeService](ctx)
	print(item)
}

func TestWithProvidedRejectsAnyReturningFactory(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok && errors.Is(err, InvalidInjectable) {
				return
			}
			t.Fatalf("Expected InvalidInjectable panic, got %v", r)
		}
		t.Fatal("Expected WithProvided to reject a func() any factory, but it did not panic")
	}()

	WithProvided(context.Background(), func() any { return &_ServeServiceImpl{} })
}

func TestWithProvidedCallableReturningNilNamedInterface(t *testing.T) {
	ctx := WithProvided(context.Background(), func() ServeService { return nil })

	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok && errors.Is(err, InvalidInjectable) {
				return
			}
			t.Fatalf("Expected InvalidInjectable panic, got %v", r)
		}
		t.Fatal("Expected panic for factory returning a nil named interface, but code did not panic")
	}()

	_ = GetProvided[ServeService](ctx)
}

// Additional test to specifically test the case where len(results) == 0
// This is a more explicit test for the scenario, even though void functions should be caught earlier
func TestWithProvidedCallableReturnsNothing(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok && errors.Is(err, InvalidInjectable) {
				// Expected behavior - functions returning nothing should be rejected
				// The error message should mention that function must return at least one value
			} else {
				t.Fatalf("Expected InvalidInjectable panic, got %v", r)
			}
		} else {
			t.Fatal("Expected panic for function returning nothing, but code did not panic")
		}
	}()

	ctx := context.Background()

	// Function that returns nothing (void function)
	voidFunc := func() {
		// Returns nothing
	}

	// This should panic at the NumOut() == 0 check
	_ = WithProvided(ctx, voidFunc)
}

// Test for the case where we have an IoC context with injectables,
// but none of them match the requested interface type
func TestGetProvidedNoMatchingInterface(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok && errors.Is(err, MissingInjectable) {
				// Expected behavior - no matching interface should panic with MissingInjectable
			} else {
				t.Fatalf("Expected MissingInjectable panic, got %v", r)
			}
		} else {
			t.Fatal("Expected panic for no matching injectable interface, but code did not panic")
		}
	}()

	ctx := context.Background()

	// Provide one type of service
	service := &_ServeServiceImpl{}
	ctx = WithProvided(ctx, service)

	// Try to retrieve a different interface type that doesn't match
	// This should trigger the len(matches) == 0 condition
	_ = GetProvided[DoSomethingService](ctx)
}

// Test that callable injectables are only invoked once (sync.OnceValue behavior)
func TestCallableInjectableInvokedOnlyOnce(t *testing.T) {
	ctx := context.Background()

	var callCount int
	factory := func() *_ServeServiceImpl {
		callCount++
		return &_ServeServiceImpl{}
	}

	// Store the factory as a lazy callable
	ctx = WithProvided(ctx, factory)

	// Retrieve the service multiple times
	service1 := GetProvided[ServeService](ctx)
	service2 := GetProvided[ServeService](ctx)
	service3 := GetProvided[ServeService](ctx)

	// Verify the factory was only called once
	if callCount != 1 {
		t.Fatalf("Expected factory to be called exactly once, but was called %d times", callCount)
	}

	// Verify we get the same instance each time
	if service1 != service2 || service2 != service3 {
		t.Fatal("Expected to get the same instance on multiple retrievals")
	}

	// Verify the service works
	if service1.Serve() != "Service is serving" {
		t.Fatal("Service does not work as expected")
	}
}

func TestIsProvidedReturnsTrue(t *testing.T) {
	var ctx = context.Background()
	ctx = WithProvided(ctx, &_ServeServiceImpl{})

	if !IsProvided[ServeService](ctx) {
		t.Fatal("expected IsProvided to return true")
	}
}

func TestIsProvidedReturnsFalse(t *testing.T) {
	var ctx = context.Background()

	if IsProvided[ServeService](ctx) {
		t.Fatal("expected IsProvided to return false")
	}
}

func TestIsProvidedReturnsFalseWhenDifferentType(t *testing.T) {
	var ctx = context.Background()
	ctx = WithProvided(ctx, &_ServeServiceImpl{})

	if IsProvided[DoSomethingService](ctx) {
		t.Fatal("expected IsProvided to return false for unregistered interface")
	}
}

func TestGetOptionalProvidedReturnsValue(t *testing.T) {
	var ctx = context.Background()
	var service = &_ServeServiceImpl{}
	ctx = WithProvided(ctx, service)

	value, found := GetOptionalProvided[ServeService](ctx)
	if !found {
		t.Fatal("expected found to be true")
	}
	if value.Serve() != "Service is serving" {
		t.Fatal("expected service to work correctly")
	}
}

func TestGetOptionalProvidedReturnsNotFound(t *testing.T) {
	var ctx = context.Background()

	value, found := GetOptionalProvided[ServeService](ctx)
	if found {
		t.Fatal("expected found to be false")
	}
	if value != nil {
		t.Fatal("expected value to be nil")
	}
}

func TestGetOptionalProvidedReturnsNotFoundWhenDifferentType(t *testing.T) {
	var ctx = context.Background()
	ctx = WithProvided(ctx, &_ServeServiceImpl{})

	value, found := GetOptionalProvided[DoSomethingService](ctx)
	if found {
		t.Fatal("expected found to be false for unregistered interface")
	}
	if value != nil {
		t.Fatal("expected value to be nil")
	}
}

func TestIsProvidedWithCallableInjectable(t *testing.T) {
	var ctx = context.Background()
	factory := func() *_ServeServiceImpl {
		return &_ServeServiceImpl{}
	}
	ctx = WithProvided(ctx, factory)

	if !IsProvided[ServeService](ctx) {
		t.Fatal("expected IsProvided to return true for callable injectable")
	}
}

func TestGetOptionalProvidedWithCallableInjectable(t *testing.T) {
	var ctx = context.Background()
	factory := func() *_ServeServiceImpl {
		return &_ServeServiceImpl{}
	}
	ctx = WithProvided(ctx, factory)

	value, found := GetOptionalProvided[ServeService](ctx)
	if !found {
		t.Fatal("expected found to be true for callable injectable")
	}
	if value.Serve() != "Service is serving" {
		t.Fatal("expected service to work correctly")
	}
}

func TestGetOptionalProvidedReturnsLatestOverride(t *testing.T) {
	var ctx = context.Background()
	ctx = WithProvided(ctx, &_ServeServiceImpl{})
	ctx = WithOverride(ctx, &AdvancedServiceImpl{})

	value, found := GetOptionalProvided[ServeService](ctx)
	if !found {
		t.Fatal("expected found to be true")
	}
	if value.Serve() != "Advanced Service is serving" {
		t.Fatal("expected the most recently provided implementation")
	}
}

type _serviceContext struct {
	context.Context
}

func TestGetProvidedFromDeepStoredContext(t *testing.T) {
	var customCtx = &_serviceContext{Context: context.Background()}
	var ctx = WithProvided(customCtx, &_DoSomethingServiceImpl{})
	ctx = WithProvided(ctx, &_DoBServiceImpl{})

	var retrieved = GetProvided[DoSomethingService](ctx)
	if retrieved.DoSomething() != "Another service is doing something" {
		t.Fatal("expected to find Service from deeply stored context")
	}
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

func TestNestedOverride(t *testing.T) {
	var ctx = context.Background()
	ctx = WithProvided(ctx, &_DoAServiceImpl{})

	var a1 = GetProvided[ServiceA](ctx)
	if a1.DoA() != "A" {
		t.Fatal("expected A")
	}

	ctx = WithProvided(ctx, &_DoBServiceImpl{})
	var a2 = GetProvided[ServiceA](ctx)
	if a2.DoA() != "A" {
		t.Fatal("expected A")
	}

	var b1 = GetProvided[ServiceB](ctx)
	if b1.DoB() != "B" {
		t.Fatal("expected B")
	}

	ctx = WithOverride(ctx, &_DoAServiceImpl2{})
	var a3 = GetProvided[ServiceA](ctx)
	if a3.DoA() != "A-Alternate" {
		t.Fatal("expected A-Alternate")
	}
}

func TestForeignStringKeyDoesNotCorruptResolution(t *testing.T) {
	var ctx = context.Background()
	ctx = WithProvided(ctx, &_ServeServiceImpl{})

	ctx = context.WithValue(ctx, "ioc", "some foreign value")

	if !IsProvided[ServeService](ctx) {
		t.Fatal("a foreign string key \"ioc\" must not shadow the ioc chain")
	}

	var retrieved = GetProvided[ServeService](ctx)
	if retrieved.Serve() != "Service is serving" {
		t.Fatal("resolution corrupted by foreign string key collision")
	}
}

func TestGetConcreteValue(t *testing.T) {
	var ctx = context.Background()
	ctx = WithProvided(ctx, &_DoAServiceImpl{})

	value, found := GetOptionalProvided[*_DoAServiceImpl](ctx)
	if !found {
		t.Fatal("expected to find concrete type")
	}
	if value.DoA() != "A" {
		t.Fatal("expected to call method on concrete type")
	}
}

func TestGetDeepConcreteValue(t *testing.T) {
	var ctx = context.Background()
	ctx = WithProvided(ctx, &_DoAServiceImpl{})
	ctx = WithProvided(ctx, &_DoBServiceImpl{})

	value, found := GetOptionalProvided[*_DoAServiceImpl](ctx)
	if !found {
		t.Fatal("expected to find concrete type in deep context")
	}
	if value.DoA() != "A" {
		t.Fatal("expected to call method on concrete type from deep context")
	}
}

func TestWithProvidedCallableReturningNilFunc(t *testing.T) {
	ctx := context.Background()

	type Handler func() string

	factory := func() Handler {
		return nil
	}

	ctx = WithProvided(ctx, factory)

	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok && errors.Is(err, InvalidInjectable) {
				// Expected: a returned nil func is uncallable and must be rejected on first resolve.
			} else {
				t.Fatalf("Expected InvalidInjectable panic, got %v", r)
			}
		} else {
			t.Fatal("Expected panic for factory returning nil func, but code did not panic")
		}
	}()

	_ = GetProvided[Handler](ctx)
}

func TestWithProvidedRejectsTypedNilPointer(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok && errors.Is(err, InvalidInjectable) {
				return
			}
			t.Fatalf("Expected InvalidInjectable panic, got %v", r)
		}
		t.Fatal("Expected WithProvided to reject a typed-nil pointer, but it did not panic")
	}()

	var nilImpl *_ServeServiceImpl = nil
	WithProvided(context.Background(), nilImpl)
}

func TestFactoryNotInvokedResolvingUnrelatedType(t *testing.T) {
	var calls int
	factory := func() *_DoSomethingServiceImpl {
		calls++
		return &_DoSomethingServiceImpl{}
	}

	ctx := WithProvided(context.Background(), factory)
	ctx = WithProvided(ctx, &_ServeServiceImpl{})

	GetProvided[ServeService](ctx)
	if calls != 0 {
		t.Fatalf("factory for an unrelated type was invoked %d times resolving ServeService, want 0", calls)
	}

	GetProvided[DoSomethingService](ctx)
	if calls != 1 {
		t.Fatalf("factory invoked %d times after resolving its own type, want 1", calls)
	}
}

func TestUnrelatedBrokenFactoryDoesNotBlockParent(t *testing.T) {
	parent := WithProvided(context.Background(), &_DoSomethingServiceImpl{})
	child := WithProvided(parent, func() ServeService { panic("unrelated factory boom") })

	if value, found := GetOptionalProvided[DoSomethingService](child); !found || value == nil {
		t.Fatal("expected parent's DoSomethingService to resolve despite an unrelated broken child factory")
	}
	if !IsProvided[DoSomethingService](child) {
		t.Fatal("expected IsProvided to be true for the parent's DoSomethingService")
	}
	if GetProvided[DoSomethingService](child).DoSomething() != "Another service is doing something" {
		t.Fatal("expected the parent's DoSomethingService instance")
	}
}

func TestWithProvidedCallableReturningTypedNilViaInterface(t *testing.T) {
	ctx := WithProvided(context.Background(), func() ServeService {
		var nilImpl *_ServeServiceImpl = nil
		return nilImpl
	})

	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok && errors.Is(err, InvalidInjectable) {
				return
			}
			t.Fatalf("Expected InvalidInjectable panic, got %v", r)
		}
		t.Fatal("Expected panic for factory returning a typed-nil pointer via an interface, but code did not panic")
	}()

	_ = GetProvided[ServeService](ctx)
}
