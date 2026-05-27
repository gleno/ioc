package ioc

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/gleno/fault"
)

type iocContextKeyType struct{}

var iocContextKey = iocContextKeyType{}

var (
	InjectionError      = fault.Sentinel("injection error")
	InvalidInjectable   = InjectionError.WithMessage("invalid injectable")
	MissingInjectable   = InjectionError.WithMessage("missing injectable")
	AmbiguousInjectable = InjectionError.WithMessage("ambiguous injectable")
	CircularInjectable  = InjectionError.WithMessage("circular injectable")
)

type _lazyInjectable struct {
	resolve func() any
	outType reflect.Type
}

func (l *_lazyInjectable) tryResolve() (value any, recovered any) {
	defer func() { recovered = recover() }()
	return l.resolve(), nil
}

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
	case *_lazyInjectable:
		return inj.outType.AssignableTo(t)
	default:
		return reflect.TypeOf(ic.injectable).AssignableTo(t)
	}
}

func resolveInjectable[T any](ic *_iocContext) (value T, recovered any) {
	switch inj := ic.injectable.(type) {
	case *_pinned:
		return inj.value.(T), nil
	case *_lazyInjectable:
		resolved, recovered := inj.tryResolve()
		if recovered != nil {
			return value, recovered
		}
		return resolved.(T), nil
	default:
		return ic.injectable.(T), nil
	}
}

func findProvided[T any](ctx context.Context) (value T, found bool, recovered any) {

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
		return value, false, nil
	case 1:
		resolved, recovered := resolveInjectable[T](live[0])
		if recovered != nil {
			return value, false, recovered
		}
		return resolved, true, nil
	default:
		panic(fault.From(AmbiguousInjectable, fmt.Sprintf("%s (%d matches)", tType, len(live))))
	}
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

func As[I any](impl I) any {
	var value = reflect.ValueOf(impl)
	if !value.IsValid() {
		panic(fault.From(InvalidInjectable, "nil"))
	}
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

	if injectableValue.Kind() == reflect.Func {
		return buildFactory(injectableValue)
	}

	if isNilInjectable(injectableValue) {
		panic(fault.From(InvalidInjectable, "nil "+injectableValue.Kind().String()))
	}

	return injectable
}

func buildFactory(injectableValue reflect.Value) *_lazyInjectable {
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

	return &_lazyInjectable{resolve: resolve, outType: outType}
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
