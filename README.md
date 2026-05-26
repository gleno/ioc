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
- **Lazy factories.** Provide a zero-arg function `func() T` and it's invoked at most once, the first time the value is resolved (`sync.OnceValue`). The same instance is returned on every subsequent resolve.
- **Most-recent-wins overrides.** Providing a second value satisfying the same interface shadows the earlier one — useful for layering defaults then overriding in tests or nested scopes.
- **The context itself can be an injectable.** If the `context.Context` you pass implements `T`, it resolves — explicit injectables take priority over this.
- **Three resolution modes.** Panic-on-missing (`GetProvided`), boolean check (`IsProvided`), and optional `(T, bool)` (`GetOptionalProvided`).
- **Typed errors.** Failures are `fault` sentinels (`MissingInjectable`, `InvalidInjectable`) you can match with `errors.Is`.

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
// (T, bool) — never panics.
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

A factory must take no parameters and return exactly one value, or `WithProvided` panics with `InvalidInjectable`. If the factory returns nil, the panic happens on first resolve, not at registration.

### Overrides: most-recent wins

When two injectables satisfy the same type, the most recently provided one is resolved. This makes layering and overriding straightforward — for example, install defaults early and override in a nested scope.

```go
ctx = ioc.WithProvided(ctx, &defaultGreeter{})
ctx = ioc.WithProvided(ctx, &customGreeter{})

g := ioc.GetProvided[Greeter](ctx) // resolves customGreeter
```

### Inheritance through derived contexts

Injectables flow through any context derived from one that provided them — including plain `context.WithValue` children.

```go
parent := ioc.WithProvided(context.Background(), &service{})
child := context.WithValue(parent, key, val)

g := ioc.GetProvided[Greeter](child) // resolves from parent
```

### The context as an injectable

If the context value itself implements the requested type, it resolves directly — no explicit `WithProvided` needed. Explicitly provided injectables take priority over the context.

```go
type serverContext struct {
	context.Context
}

func (c *serverContext) Greet() string { return "from context" }

ctx := &serverContext{Context: context.Background()}

g := ioc.GetProvided[Greeter](ctx) // resolves the context itself
```

## API

```go
// Resolve T, or panic with MissingInjectable if absent.
func GetProvided[T any](ctx context.Context) T

// Resolve T, returning whether it was found.
func GetOptionalProvided[T any](ctx context.Context) (T, bool)

// Report whether a value satisfying T is available.
func IsProvided[T any](ctx context.Context) bool

// Return a context carrying the given injectables (values or zero-arg factory funcs).
func WithProvided[TContext context.Context](ctx TContext, injectables ...any) context.Context
```

### Errors

```go
var (
	InjectionError    // base sentinel for all injection failures
	InvalidInjectable // nil injectable, or a malformed factory func
	MissingInjectable // no provided value satisfies the requested type
)
```

Both `InvalidInjectable` and `MissingInjectable` derive from `InjectionError`, so `errors.Is(err, ioc.InjectionError)` matches either. Failures surface as panics; recover and match with `errors.Is`.

## License

See [LICENSE](LICENSE).
