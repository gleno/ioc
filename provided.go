package ioc

import (
	"context"
	"fmt"
	"reflect"
	"slices"

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
	value any
	types []reflect.Type
}

type _iocContext struct {
	parent     *_iocContext
	injectable any
	override   bool
}

func (ic *_iocContext) matches(t reflect.Type) bool {
	switch inj := ic.injectable.(type) {
	case *_pinned:
		return slices.Contains(inj.types, t)
	default:
		return reflect.TypeOf(ic.injectable).AssignableTo(t)
	}
}

func resolveInjectable[T any](ic *_iocContext) T {
	switch inj := ic.injectable.(type) {
	case *_pinned:
		return inj.value.(T)
	default:
		return ic.injectable.(T)
	}
}

func findProvided[T any](ctx context.Context) (value T, found bool) {
	var tType = reflect.TypeFor[T]()
	if tType.Kind() == reflect.Interface && tType.NumMethod() == 0 {
		panic(fault.From(InvalidInjectable, "cannot resolve interface{}"))
	}

	iocContext, _ := ctx.Value(iocContextKey).(*_iocContext)

	var live []*_iocContext
	for ic := iocContext; ic != nil; ic = ic.parent {
		if !ic.matches(tType) {
			continue
		}
		live = append(live, ic)
		if ic.override {
			break
		}
	}

	switch len(live) {
	case 0:
		return value, false
	case 1:
		return resolveInjectable[T](live[0]), true
	default:
		panic(fault.From(AmbiguousInjectable, fmt.Sprintf("%s (%d matches)", tType, len(live))))
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
	var value = reflect.ValueOf(impl)
	if !value.IsValid() {
		panic(fault.From(InvalidInjectable, "nil"))
	}
	rejectFuncInjectable(value)
	if isNilInjectable(value) {
		panic(fault.From(InvalidInjectable, "nil "+value.Kind().String()))
	}
	return &_pinned{value: impl, types: []reflect.Type{reflect.TypeFor[I]()}}
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
		iocContext = &_iocContext{
			parent:     iocContext,
			injectable: buildInjectable(injectable),
			override:   override,
		}
	}
	return context.WithValue(ctx, iocContextKey, iocContext)
}

func buildInjectable(injectable any) any {
	if injectable == nil {
		panic(fault.From(InvalidInjectable, "nil"))
	}

	if _, ok := injectable.(*_pinned); ok {
		return injectable
	}

	var injectableValue = reflect.ValueOf(injectable)
	rejectFuncInjectable(injectableValue)
	if isNilInjectable(injectableValue) {
		panic(fault.From(InvalidInjectable, "nil "+injectableValue.Kind().String()))
	}

	return injectable
}

func rejectFuncInjectable(v reflect.Value) {
	if v.Kind() != reflect.Func {
		return
	}
	if v.IsNil() {
		panic(fault.From(InvalidInjectable, "nil function"))
	}
	panic(fault.From(InvalidInjectable, "function"))
}

func isNilInjectable(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Chan:
		return v.IsNil()
	}
	return false
}
