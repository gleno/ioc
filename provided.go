package ioc

import (
	"context"
	"fmt"
	"reflect"

	"github.com/gleno/fault"
)

type iocContextKeyType struct{}

var iocContextKey = iocContextKeyType{}

var (
	InjectionError      = fault.Sentinel("injection error")
	InvalidInjectable   = InjectionError.WithMessage("invalid injectable")
	MissingInjectable   = InjectionError.WithMessage("missing injectable")
	AmbiguousInjectable = InjectionError.WithMessage("ambiguous injectable")
)

type _pinned struct {
	value  any
	target reflect.Type
}

type _iocContext struct {
	parent     *_iocContext
	injectable any
	override   bool
}

func findProvided[T any](ctx context.Context) (value T, found bool) {
	var tType = reflect.TypeFor[T]()
	validateResolvableType(tType)

	iocContext, _ := ctx.Value(iocContextKey).(*_iocContext)

	var matches int
	for ic := iocContext; ic != nil; ic = ic.parent {
		injectable := ic.injectable
		if pinned, ok := injectable.(*_pinned); ok {
			if pinned.target != tType {
				continue
			}
			injectable = pinned.value
		}
		candidate, ok := injectable.(T)
		if !ok {
			continue
		}
		if matches == 0 {
			value = candidate
		}
		matches++
		if ic.override {
			break
		}
	}

	switch matches {
	case 0:
		return value, false
	case 1:
		return value, true
	default:
		panic(fault.From(AmbiguousInjectable, fmt.Sprintf("%s (%d matches)", tType, matches)))
	}
}

func GetProvided[T any](ctx context.Context) T {
	value, found := findProvided[T](ctx)
	if !found {
		panic(fault.From(MissingInjectable, reflect.TypeFor[T]().String()))
	}
	return value
}

func IsProvided[T any](ctx context.Context) bool {
	_, found := findProvided[T](ctx)
	return found
}

func GetOptionalProvided[T any](ctx context.Context) (T, bool) {
	value, found := findProvided[T](ctx)
	return value, found
}

func As[I any](impl I) any {
	validateInjectable(impl)
	target := reflect.TypeFor[I]()
	validateResolvableType(target)
	return &_pinned{value: impl, target: target}
}

func WithProvided[TContext context.Context](ctx TContext, injectables ...any) context.Context {
	return withInjectables(ctx, false, injectables)
}

func WithOverride[TContext context.Context](ctx TContext, injectables ...any) context.Context {
	return withInjectables(ctx, true, injectables)
}

func withInjectables(ctx context.Context, override bool, injectables []any) context.Context {
	iocContext, _ := ctx.Value(iocContextKey).(*_iocContext)
	for _, injectable := range injectables {
		validateInjectable(injectable)
		iocContext = &_iocContext{
			parent:     iocContext,
			injectable: injectable,
			override:   override,
		}
	}
	return context.WithValue(ctx, iocContextKey, iocContext)
}

func validateInjectable(injectable any) {
	value := reflect.ValueOf(injectable)
	if !value.IsValid() {
		panic(fault.From(InvalidInjectable, "nil"))
	}

	switch value.Kind() {
	case reflect.Func:
		if value.IsNil() {
			panic(fault.From(InvalidInjectable, "nil function"))
		}
		panic(fault.From(InvalidInjectable, "function"))
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Chan, reflect.UnsafePointer:
		if value.IsNil() {
			panic(fault.From(InvalidInjectable, "nil "+value.Kind().String()))
		}
	}
}

func validateResolvableType(target reflect.Type) {
	if target.Kind() == reflect.Interface && target.NumMethod() == 0 {
		panic(fault.From(InvalidInjectable, "cannot resolve interface{}"))
	}
}
