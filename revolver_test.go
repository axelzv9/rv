package rv

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/axelzv9/rv/testdata/test"
	test2 "github.com/axelzv9/rv/testdata/test/test"
)

func TestRevolve(t *testing.T) {
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			called := false

			err := Revolve(ctx, WithStdLogger(), WithDebug(), testCase.Option, Invoke(func() {
				called = true
			}))
			if err != testCase.Error {
				if !(err == nil || testCase.Error == nil) {
					if errors.Is(err, testCase.Error) {
						return
					}
				}
				t.Fatalf("errors are not equal: \ngot: %v \nexp: %v", err, testCase.Error)
			}
			if called == testCase.InvokeMustBeSkipped {
				t.Fatal("invoke func must be called")
			}
		})
	}
}

type TestCase struct {
	Option              Option
	Error               error
	InvokeMustBeSkipped bool
}

var testCases = map[string]TestCase{
	"dry run": {
		Option: nil,
		Error:  nil,
	},
	"lazy init": {
		Option: Provide(func() *Foo {
			panic("it must not be called")
		}),
		Error: nil,
	},
	"unordered": {
		Option: Options(
			Invoke(func(foo *Foo) {
				if foo == nil {
					panic("foo must not be nil")
				}
			}),
			Provide(func() *Foo { return &Foo{} }),
		),
		Error: nil,
	},
	"provide unsupported": {
		Option: Options(
			Provide(&Foo{}),
			Invoke(func(foo *Foo) {
				if foo == nil {
					panic("foo must not be nil")
				}
			}),
		),
		Error: ErrUnsupportedProvideTarget,
	},
	"invoke unsupported": {
		Option: Options(
			Provide(func() *Foo { return &Foo{} }),
			Invoke(&Foo{}),
		),
		Error: ErrUnsupportedInvokeTarget,
	},
	"with logger": {
		Option: Options(
			Provide(func() *Foo { return &Foo{} }),
			Invoke(func(foo *Foo) {
				if foo == nil {
					panic("foo must not be nil")
				}
			}),
			WithStdLogger(),
		),
		Error: nil,
	},
	"provide error": {
		Option: Options(
			Provide(func() (*Foo, error) { return nil, provideTestError }),
			Invoke(func(foo *Foo) {
				if foo == nil {
					panic("foo must not be nil")
				}
			}),
		),
		Error:               provideTestError,
		InvokeMustBeSkipped: true,
	},
	"invoke error": {
		Option: Options(
			Provide(func() *Foo { return &Foo{} }),
			Invoke(func(foo *Foo) error {
				if foo == nil {
					panic("foo must not be nil")
				}
				return invokeTestError
			}),
		),
		Error:               invokeTestError,
		InvokeMustBeSkipped: true,
	},
	"provide with dependency": {
		Option: Options(
			Provide(func(foo Foo) *Foo {
				return &foo
			}, func(bar *Bar) Foo {
				if bar == nil {
					panic("bar must not be nil")
				}
				return Foo{}
			}, func() *Bar { return &Bar{} }),
			Invoke(func(foo *Foo) {
				if foo == nil {
					panic("foo must not be nil")
				}
			}),
		),
		Error: nil,
	},
	"similar package names": {
		Option: Options(
			Provide(test.NewBar, test2.NewBar),
			Invoke(func(bar1 *test.Bar, bar2 *test2.Bar) {
				if bar1 == nil {
					panic("bar must not be nil")
				}
				if bar2 == nil {
					panic("bar2 must not be nil")
				}
			}),
		),
		Error: nil,
	},
	"multiple provide": {
		Option: Options(
			Provide(func() *Foo { return &Foo{} }, func() *Foo { return &Foo{} }),
			Invoke(func(foo *Foo) {
				if foo == nil {
					panic("foo must not be nil")
				}
			}),
		),
		Error:               ErrMultipleProvide,
		InvokeMustBeSkipped: true,
	},
	"supply": {
		Option: Options(
			Supply(&Foo{}),
			Invoke(func(foo *Foo) {
				if foo == nil {
					panic("foo must not be nil")
				}
			}, func(foo *Foo) {
				if foo == nil {
					panic("foo must not be nil")
				}
			}),
		),
	},
	"supply context": {
		Option: Options(
			WithDuckTyping(),
			Supply(context.Background()),
			Invoke(func(ctx context.Context) {
				// on the first time ctx will be provided directly
				if ctx == nil {
					panic("ctx must not be nil")
				}
			}, func(ctx context.Context) {
				// on the second time ctx will be provided from instances pool
				if ctx == nil {
					panic("ctx must not be nil")
				}
			}),
		),
	},
	"duck typing": {
		Option: Options(
			WithDuckTyping(),
			Supply(&FooBar{}),
			Invoke(func(foo IFoo) {
				if foo == nil {
					panic("foo must not be nil")
				}
			}, func(bar IBar) {
				if bar == nil {
					panic("bar must not be nil")
				}
			}),
		),
	},
	"duck typing multiple provide": {
		Option: Options(
			WithDuckTyping(),
			Supply(&FooBar{}, &Foo{}),
			Invoke(func(foo IFoo) {
				if foo == nil {
					panic("foo must not be nil")
				}
			}, func(bar IBar) {
				if bar == nil {
					panic("bar must not be nil")
				}
			}),
		),
		Error:               ErrMultipleProvide,
		InvokeMustBeSkipped: true,
	},
	"cyclic_provide": {
		Option: Options(
			Provide(
				func(*Foo) *Bar {
					return &Bar{}
				},
				func(*Bar) *Foo {
					return &Foo{}
				},
			),
			Invoke(func(foo *Foo, bar *Bar) {
				if foo == nil {
					panic("foo must not be nil")
				}
				if bar == nil {
					panic("bar must not be nil")
				}
			}),
		),
		Error:               ErrCyclicProvideDetected,
		InvokeMustBeSkipped: true,
	},
}

type Foo struct{}

func (Foo) foo() {}

type Bar struct{}

func (Bar) bar() {}

type IFoo interface {
	foo()
}

type IBar interface {
	bar()
}

type FooBar struct{}

func (FooBar) foo() {}
func (FooBar) bar() {}

var provideTestError = errors.New("provide test err")
var invokeTestError = errors.New("invoke test err")
