package ioc

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/gleno/fault"
)

type iocContextKeyType struct{}

var iocContextKey = iocContextKeyType{}

var (
	InjectionError    = fault.Sentinel("injection error")
	InvalidInjectable = InjectionError.WithMessage("invalid injectable")
	MissingInjectable = InjectionError.WithMessage("missing injectable")
)

type _lazyInjectable struct {
	value   func() any
	outType reflect.Type
}

type _iocContext struct {
	parent     *_iocContext
	injectable any
}

func findProvided[T any](ctx context.Context) (value T, found bool) {

	iocContext, _ := ctx.Value(iocContextKey).(*_iocContext)
	var tType = reflect.TypeFor[T]()

	for ic := iocContext; ic != nil; ic = ic.parent {
		var v = ic.injectable
		if lazy, ok := v.(*_lazyInjectable); ok {
			if !lazy.outType.AssignableTo(tType) {
				continue
			}
			v = lazy.value()
		}
		if matched, ok := v.(T); ok {
			return matched, true
		}
	}

	return
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
	return findProvided[T](ctx)
}

func WithProvided[TContext context.Context](ctx TContext, injectables ...any) context.Context {
	iocContext, _ := ctx.Value(iocContextKey).(*_iocContext)
	for _, injectable := range injectables {

		if injectable == nil {
			panic(fault.From(InvalidInjectable, "nil"))
		}

		var injectableValue = reflect.ValueOf(injectable)

		if injectableValue.Kind() == reflect.Func {
			var funcType = injectableValue.Type()

			if injectableValue.IsNil() {
				panic(fault.From(InvalidInjectable, "nil function"))
			}

			if funcType.NumIn() > 0 {
				panic(fault.From(InvalidInjectable, "function cannot have parameters"))
			}

			if funcType.NumOut() != 1 {
				panic(fault.From(InvalidInjectable, fmt.Sprintf("function must return one value, but has %d ", funcType.NumOut())))
			}

			var outType = funcType.Out(0)
			if outType.Kind() == reflect.Interface && outType.NumMethod() == 0 {
				panic(fault.From(InvalidInjectable, "factory must return a concrete type, not interface{}"))
			}

			var factory = sync.OnceValue(func() any {

				var result = injectableValue.Call(nil)[0]
				var produced = result
				if produced.Kind() == reflect.Interface {
					if produced.IsNil() {
						panic(fault.From(InvalidInjectable, "function returned nil"))
					}
					produced = produced.Elem()
				}
				if (produced.Kind() == reflect.Ptr || produced.Kind() == reflect.Func) && produced.IsNil() {
					panic(fault.From(InvalidInjectable, "function returned nil"))
				}

				return result.Interface()
			})

			iocContext = &_iocContext{parent: iocContext, injectable: &_lazyInjectable{value: factory, outType: outType}}
		} else {
			if injectableValue.Kind() == reflect.Ptr && injectableValue.IsNil() {
				panic(fault.From(InvalidInjectable, "nil pointer"))
			}
			iocContext = &_iocContext{parent: iocContext, injectable: injectable}
		}
	}

	return context.WithValue(ctx, iocContextKey, iocContext)
}
