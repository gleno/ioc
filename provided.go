package ioc

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/gleno/fault"
)

type iocContextKeyType struct{}

var iocContextKey = iocContextKeyType{}

var (
	InjectionError     = fault.Sentinel("injection error")
	InvalidInjectable  = InjectionError.WithMessage("invalid injectable")
	MissingInjectable  = InjectionError.WithMessage("missing injectable")
	CircularInjectable = InjectionError.WithMessage("circular injectable")
)

type _lazyInjectable struct {
	resolve func() any
	outType reflect.Type
}

func (l *_lazyInjectable) tryResolve() (value any, recovered any) {
	defer func() { recovered = recover() }()
	return l.resolve(), nil
}

type _iocContext struct {
	parent     *_iocContext
	injectable any
}

func findProvided[T any](ctx context.Context) (value T, found bool, recovered any) {

	var tType = reflect.TypeFor[T]()
	if tType.Kind() == reflect.Interface && tType.NumMethod() == 0 {
		panic(fault.From(InvalidInjectable, "cannot resolve interface{}"))
	}

	iocContext, _ := ctx.Value(iocContextKey).(*_iocContext)

	for ic := iocContext; ic != nil; ic = ic.parent {
		var v = ic.injectable
		if lazy, ok := v.(*_lazyInjectable); ok {
			if !lazy.outType.AssignableTo(tType) {
				continue
			}
			if v, recovered = lazy.tryResolve(); recovered != nil {
				return value, false, recovered
			}
		}
		if matched, ok := v.(T); ok {
			return matched, true, nil
		}
	}

	return value, false, nil
}

func GetProvided[T any](ctx context.Context) T {
	value, found, recovered := findProvided[T](ctx)
	if recovered != nil {
		panic(recovered)
	}
	if !found {
		panic(fault.From(MissingInjectable, reflect.TypeFor[T]().String()))
	}
	return value
}

func IsProvided[T any](ctx context.Context) bool {
	_, found, _ := findProvided[T](ctx)
	return found
}

func GetOptionalProvided[T any](ctx context.Context) (T, bool) {
	value, found, _ := findProvided[T](ctx)
	return value, found
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

			var owner atomic.Int64
			var once = sync.OnceValue(func() any {
				owner.Store(goid())
				defer owner.Store(0)

				var result = injectableValue.Call(nil)[0]
				var produced = result
				if produced.Kind() == reflect.Interface {
					if produced.IsNil() {
						panic(fault.From(InvalidInjectable, "function returned nil"))
					}
					produced = produced.Elem()
				}
				if isNilInjectable(produced) {
					panic(fault.From(InvalidInjectable, "function returned nil"))
				}

				return result.Interface()
			})

			var resolve = func() any {
				if o := owner.Load(); o != 0 && o == goid() {
					panic(fault.From(CircularInjectable, outType.String()))
				}
				return once()
			}

			iocContext = &_iocContext{parent: iocContext, injectable: &_lazyInjectable{resolve: resolve, outType: outType}}
		} else {
			if isNilInjectable(injectableValue) {
				panic(fault.From(InvalidInjectable, "nil "+injectableValue.Kind().String()))
			}
			iocContext = &_iocContext{parent: iocContext, injectable: injectable}
		}
	}

	return context.WithValue(ctx, iocContextKey, iocContext)
}

func isNilInjectable(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Pointer, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return v.IsNil()
	}
	return false
}

func goid() int64 {
	var buf [32]byte
	var s = buf[:runtime.Stack(buf[:], false)]
	s = s[len("goroutine "):]
	var id int64
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		id = id*10 + int64(c-'0')
	}
	return id
}
