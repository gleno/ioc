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
type panicContextKeyType struct{}

var iocContextKey = iocContextKeyType{}
var panicContextKey = panicContextKeyType{}

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

// PanicHandler handles a panic recovered while resolving a lazy factory.
// Return true to treat that factory as absent, or false to let older handlers try.
type PanicHandler func(any) bool

type _panicContext struct {
	parent  *_panicContext
	handler PanicHandler
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
		resolved, recovered := resolveInjectable[T](live[0])
		if recovered != nil {
			if !handleRecoveredPanic(ctx, recovered) {
				panic(recovered)
			}
			return value, false
		}
		return resolved, true
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

// OnPanic returns a context with a handler for panics recovered while resolving lazy factories.
func OnPanic(ctx context.Context, handler PanicHandler) context.Context {
	if handler == nil {
		panic(fault.From(InvalidInjectable, "nil panic handler"))
	}
	panicContext, _ := ctx.Value(panicContextKey).(*_panicContext)
	return context.WithValue(ctx, panicContextKey, &_panicContext{
		parent:  panicContext,
		handler: handler,
	})
}

// SilencePanics returns a context that treats all lazy factory panics as handled.
func SilencePanics(ctx context.Context) context.Context {
	return OnPanic(ctx, func(any) bool {
		return true
	})
}

func handleRecoveredPanic(ctx context.Context, recovered any) bool {
	panicContext, _ := ctx.Value(panicContextKey).(*_panicContext)
	for pc := panicContext; pc != nil; pc = pc.parent {
		if pc.handler(recovered) {
			return true
		}
	}
	return false
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
