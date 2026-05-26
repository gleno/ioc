package ioc

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/gleno/fault"
)

const (
	iocContextKey = "ioc"
)

var (
	InjectionError    = fault.Sentinel("injection error")
	InvalidInjectable = InjectionError.WithMessage("invalid injectable")
	MissingInjectable = InjectionError.WithMessage("missing injectable")
)

type _iocContext struct {
	parent     *_iocContext
	injectable any
	// For lazy callables, we store the sync.OnceValue func
	lazyFactory func() any
}

func findProvided[T any](ctx context.Context) (value T, found bool) {

	iocContext, _ := ctx.Value(iocContextKey).(*_iocContext)

	for ic := iocContext; ic != nil; ic = ic.parent {
		var v any
		if ic.lazyFactory != nil {
			v = ic.lazyFactory()
		} else {
			v = ic.injectable
		}
		if matched, ok := v.(T); ok {
			return matched, true
		}
	}

	if matched, ok := ctx.(T); ok {
		return matched, true
	}

	return
}

func GetProvided[T any](ctx context.Context) T {
	value, found := findProvided[T](ctx)
	if !found {
		var tType = reflect.TypeOf((*T)(nil)).Elem()
		panic(fault.From(MissingInjectable, tType.String()))
	}
	return value
}

func IsProvided[T any](ctx context.Context) bool {
	_, found := findProvided[T](ctx)
	return found
}

func GetOptionalProvided[T any](ctx context.Context) (T, bool) {
	return findProvided[T](ctx)
}

func WithProvided[TContext context.Context](ctx TContext, injectables ...any) context.Context {
	iocContext, _ := ctx.Value(iocContextKey).(*_iocContext)
	for _, injectable := range injectables {

		if injectable == nil {
			panic(fault.From(InvalidInjectable, "nil"))
		}

		// injectable could be callable?
		if reflect.TypeOf(injectable).Kind() == reflect.Func {
			var funcValue = reflect.ValueOf(injectable)
			var funcType = funcValue.Type()

			// Check if function has parameters
			if funcType.NumIn() > 0 {
				panic(fault.From(InvalidInjectable, "function cannot have parameters"))
			}

			if funcType.NumOut() != 1 {
				panic(fault.From(InvalidInjectable, fmt.Sprintf("function must return one value, but has %d ", funcType.NumOut())))
			}

			var lazyFactory = sync.OnceValue(func() any {

				var results = funcValue.Call(nil)

				// Use the first return value (we know results has at least one element due to NumOut() > 0 check above)
				var result = results[0]
				if result.Kind() == reflect.Ptr && result.IsNil() {
					panic(fault.From(InvalidInjectable, "function returned nil"))
				}

				var value = result.Interface()
				if value == nil {
					panic(fault.From(InvalidInjectable, "function returned nil"))
				}

				return value
			})

			// Store as lazy callable
			iocContext = &_iocContext{parent: iocContext, lazyFactory: lazyFactory}
		} else {
			iocContext = &_iocContext{parent: iocContext, injectable: injectable}
		}
	}

	return context.WithValue(ctx, iocContextKey, iocContext)
}
