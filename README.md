# ioc

A lightweight inversion-of-control / dependency-injection container for Go, scoped entirely to `context.Context`. Dependencies are attached to a context, resolved by interface (or concrete) type via generics, and inherited by every derived context — no global registry, no wiring, no container object to thread around.

## Install

```sh
go get github.com/gleno/ioc
```

## Features

- **Context-scoped.** Injectables live on a `context.Context`. Anything that has the context can resolve them; child contexts inherit everything from their parents.
- **Resolve by type.** `GetProvided[T]` matches a stored value against the requested type `T` — interface or concrete — using Go's normal type assertion. One implementation can satisfy many interfaces.
- **Generic, no reflection at the call site.** Resolution is type-safe; you get a `T` back, not an `any` to cast.
- **Lazy factories.** Provide a zero-arg function `func() T` and it's invoked at most once — the first time a `T` is resolved (`sync.OnceValue`), and never as a side effect of resolving an unrelated type. A factory resolves as its declared return type `T`, which must be a concrete or named-interface type (not `interface{}`).
- **Configurable factory panic handling.** Lazy factory panics are loud by default, but a context can opt into handling them with `OnPanic` or silencing them with `SilencePanics`.
- **Ambiguity is loud.** If two providers satisfy the requested type, resolution **panics** rather than silently picking one. Replace a provider deliberately with `WithOverride`; pin a provider to exact types with `As`.
- **Three resolution modes.** Panic-on-missing (`GetProvided`), boolean check (`IsProvided`), and optional `(T, bool)` (`GetOptionalProvided`).
- **Typed errors.** Failures are `fault` sentinels (`MissingInjectable`, `InvalidInjectable`, `AmbiguousInjectable`, `CircularInjectable`) you can match with `errors.Is`.

## Usage

### Provide and resolve

`WithProvided` returns a new context carrying the injectable. `GetProvided[T]` resolves it by type — here a concrete `*service` is registered and resolved as the `Greeter` interface it implements.

```go
package main

import (
	"context"
	"fmt"

	"github.com/gleno/ioc"
)

type Greeter interface {
	Greet() string
}

type service struct{}

func (s *service) Greet() string { return "hello" }

func main() {
	ctx := context.Background()
	ctx = ioc.WithProvided(ctx, &service{})

	g := ioc.GetProvided[Greeter](ctx)
	fmt.Println(g.Greet()) // hello
}
```

`GetProvided[T]` **panics** with `MissingInjectable` if no value satisfies `T`. Use it when the dependency is required.

### Optional resolution and existence checks

```go
// (T, bool) — never panics on absence (but does panic on ambiguity; see below).
g, ok := ioc.GetOptionalProvided[Greeter](ctx)
if ok {
	fmt.Println(g.Greet())
}

// Just a presence check.
if ioc.IsProvided[Greeter](ctx) {
	// ...
}
```

### Providing multiple injectables

Pass several at once, or in separate `WithProvided` calls — both compose into the same chain.

```go
ctx = ioc.WithProvided(ctx, &serviceA{}, &serviceB{})

// equivalently:
ctx = ioc.WithProvided(ctx, &serviceA{})
ctx = ioc.WithProvided(ctx, &serviceB{})

a := ioc.GetProvided[ServiceA](ctx)
b := ioc.GetProvided[ServiceB](ctx)
```

### Lazy factories

Provide a zero-argument function returning the value. It's evaluated lazily on first resolve and cached — every later resolve returns the same instance.

```go
ctx = ioc.WithProvided(ctx, func() Greeter {
	return &service{} // constructed once, on first GetProvided
})

g := ioc.GetProvided[Greeter](ctx) // factory runs here
```

A factory must take no parameters and return exactly one value of a concrete or named-interface type, or `WithProvided` panics with `InvalidInjectable` — a `func() any` is rejected at registration, since its declared type carries no resolution information. A factory resolves only as its declared return type, and is invoked only when that type is requested. If the factory returns nil, the panic happens on first resolve, not at registration.

### Handling factory panics

Lazy factory panics propagate by default. To treat a factory panic as absence for a subtree of contexts, install a handler with `OnPanic`. Handlers run newest-to-oldest; returning `true` handles the panic, and returning `false` lets older handlers try.

```go
ctx = ioc.OnPanic(ctx, func(recovered any) bool {
	err, ok := recovered.(error)
	return ok && errors.Is(err, transientStartupFailure)
})
```

When a factory panic is handled, `GetOptionalProvided[T]` returns `(zero, false)`, `IsProvided[T]` returns `false`, and `GetProvided[T]` panics with `MissingInjectable`. To handle every lazy factory panic, use `SilencePanics`.

```go
ctx = ioc.SilencePanics(ctx)
```

`OnPanic` only handles panics recovered from lazy factory resolution. Wiring errors such as ambiguous providers, invalid registration, missing required dependencies, and resolving `any` still panic normally.

### Ambiguity and overrides

`WithProvided` asserts that what you add is *the* provider for the types it satisfies. If two `WithProvided` values satisfy the same requested type, resolving it **panics** with `AmbiguousInjectable` — a wrong match is never silent. To replace a provider on purpose, use `WithOverride`: it shadows older providers it overlaps with, per resolved type, and the newest override wins.

```go
ctx = ioc.WithProvided(ctx, &defaultGreeter{})
ctx = ioc.WithOverride(ctx, &customGreeter{}) // intentionally shadows the default

g := ioc.GetProvided[Greeter](ctx) // resolves customGreeter

// Two plain providers for the same interface, on the other hand, panic:
ctx = ioc.WithProvided(ctx, &defaultGreeter{}, &customGreeter{})
ioc.GetProvided[Greeter](ctx) // panics AmbiguousInjectable
```

Ambiguity is a wiring error, not an absence, so it panics in **all three** resolvers — including `GetOptionalProvided` and `IsProvided`.

### Pinning with `As`

`As[I](impl)` registers `impl` to resolve **only** as the exact type `I`. The compiler rejects an `impl` that doesn't implement `I`, and at resolve time `impl` matches `I` and nothing else — not a sub-interface of `I`, not some other interface `impl` happens to satisfy.

```go
// emailService has Send() and Compose(); without pinning it would also resolve
// as io.Closer, fmt.Stringer, or any other shape it incidentally satisfies.
ctx = ioc.WithProvided(ctx, ioc.As[Emailer](&emailService{}))

ioc.GetProvided[Emailer](ctx)  // resolves
ioc.IsProvided[Notifier](ctx)  // false — pinned to Emailer only
```

Pinning is opt-in: plain `WithProvided` keeps full structural resolution. Reach for `As` when you want a provider to resolve as exactly the types you name — including third-party interfaces you can't otherwise constrain.

### Avoiding accidental matches

Resolution is structural: a value matches every interface its method set covers. Two levers keep that from matching something you didn't intend:

- **`As[I](impl)` — per provider.** Pin a specific impl to exact types (above). Works on any interface, including ones you don't own.
- **Branded interfaces — per interface.** Give a domain interface an unexported marker method so nothing satisfies it by accident, across all providers.

```go
type Greeter interface {
	Greet() string
	greeter() // unexported — only your impls can satisfy it
}
```

### Inheritance through derived contexts

Injectables flow through any context derived from one that provided them — including plain `context.WithValue` children.

```go
parent := ioc.WithProvided(context.Background(), &service{})
child := context.WithValue(parent, key, val)

g := ioc.GetProvided[Greeter](child) // resolves from parent
```

## API

```go
// Resolve T, or panic with MissingInjectable if absent.
func GetProvided[T any](ctx context.Context) T

// Resolve T, returning whether it was found.
func GetOptionalProvided[T any](ctx context.Context) (T, bool)

// Report whether a value satisfying T is available.
func IsProvided[T any](ctx context.Context) bool

// Handle panics recovered while resolving lazy factories.
type PanicHandler func(any) bool
func OnPanic(ctx context.Context, handler PanicHandler) context.Context
func SilencePanics(ctx context.Context) context.Context

// Return a context carrying the given injectables (values, zero-arg factory funcs, or As pins).
// Resolving a type satisfied by two of them panics with AmbiguousInjectable.
func WithProvided[TContext context.Context](ctx TContext, injectables ...any) context.Context

// Like WithProvided, but each injectable shadows older providers it overlaps with.
func WithOverride[TContext context.Context](ctx TContext, injectables ...any) context.Context

// Pin impl to resolve only as the exact type I. Pass the result to WithProvided/WithOverride.
func As[I any](impl I) any
```

### Errors

```go
var (
	InjectionError      // base sentinel for all injection failures
	InvalidInjectable   // nil injectable (incl. typed-nil pointer/func), or a malformed factory
	MissingInjectable   // no provided value satisfies the requested type
	AmbiguousInjectable // two providers satisfy the requested type
	CircularInjectable  // a factory depends on itself (directly or through a cycle)
)
```

All four derive from `InjectionError`, so `errors.Is(err, ioc.InjectionError)` matches any of them. Failures surface as panics; recover and match with `errors.Is`.

## License

See [LICENSE](LICENSE).
