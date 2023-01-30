package rv

import (
	"context"
	"errors"
	"log"
	"testing"
	"time"

	"github.com/axelzv9/rv/testdata/test"
	test2 "github.com/axelzv9/rv/testdata/test/test"
)

func TestRevolve(t *testing.T) {
	testCases := []struct {
		name                string
		option              Option
		error               error
		invokeMustBeSkipped bool
	}{
		{
			name:   "empty run",
			option: nil,
			error:  nil,
		},
		{
			name: "lazy init",
			option: Provide(func() *Foo {
				panic("it must not be called")
			}),
			error: nil,
		},
		{
			name: "unordered",
			option: Options(
				Invoke(func(foo *Foo) {
					if foo == nil {
						panic("foo must not be nil")
					}
				}),
				Provide(func() *Foo { return &Foo{} }),
			),
			error: nil,
		},
		{
			name: "dry run unordered",
			option: Options(
				WithDryRun(),
				Invoke(func(foo *Foo) {
					if foo == nil {
						panic("foo must not be nil")
					}
				}),
				Provide(func() *Foo { return &Foo{} }),
			),
			error:               nil,
			invokeMustBeSkipped: true,
		},
		{
			name: "provide unsupported",
			option: Options(
				Provide(&Foo{}),
				Invoke(func(foo *Foo) {
					if foo == nil {
						panic("foo must not be nil")
					}
				}),
			),
			error: ErrUnsupportedProvideTarget,
		},
		{
			name: "dry run provide unsupported",
			option: Options(
				WithDryRun(),
				Provide(&Foo{}),
				Invoke(func(foo *Foo) {
					if foo == nil {
						panic("foo must not be nil")
					}
				}),
			),
			error: ErrUnsupportedProvideTarget,
		},
		{
			name: "invoke unsupported",
			option: Options(
				Provide(func() *Foo { return &Foo{} }),
				Invoke(&Foo{}),
			),
			error: ErrUnsupportedInvokeTarget,
		},
		{
			name: "dry run invoke unsupported",
			option: Options(
				WithDryRun(),
				Provide(func() *Foo { return &Foo{} }),
				Invoke(&Foo{}),
			),
			error: ErrUnsupportedInvokeTarget,
		},
		{
			name: "with log func",
			option: Options(
				Provide(func() *Foo { return &Foo{} }),
				Invoke(func(foo *Foo) {
					if foo == nil {
						panic("foo must not be nil")
					}
				}),
				WithLogger(LogFunc(customLogFunc)),
			),
			error: nil,
		},
		{
			name: "with custom log func",
			option: Options(
				Provide(func() *Foo { return &Foo{} }),
				Invoke(func(foo *Foo) {
					if foo == nil {
						panic("foo must not be nil")
					}
				}),
				WithLogger(customLogFunc),
			),
			error: nil,
		},
		{
			name: "with logger",
			option: Options(
				Provide(func() *Foo { return &Foo{} }),
				Invoke(func(foo *Foo) {
					if foo == nil {
						panic("foo must not be nil")
					}
				}),
				WithLogger(Logger(customLogger{})),
			),
			error: nil,
		},
		{
			name: "with custom logger",
			option: Options(
				Provide(func() *Foo { return &Foo{} }),
				Invoke(func(foo *Foo) {
					if foo == nil {
						panic("foo must not be nil")
					}
				}),
				WithLogger(customLogger{}),
			),
			error: nil,
		},
		{
			name: "with logger func",
			option: Options(
				Provide(func() *Foo { return &Foo{} }),
				Invoke(func(foo *Foo) {
					if foo == nil {
						panic("foo must not be nil")
					}
				}),
				WithLogger(func() Logger {
					return customLogger{}
				}),
			),
			error: nil,
		},
		{
			name: "with logger func with dependency",
			option: Options(
				Provide(func() *Foo { return &Foo{} }),
				Invoke(func(foo *Foo) {
					if foo == nil {
						panic("foo must not be nil")
					}
				}),
				WithLogger(func(foo *Foo) Logger {
					if foo == nil {
						panic("foo must not be nil")
					}
					return customLogger{}
				}),
			),
			error: nil,
		},
		{
			name: "provide error",
			option: Options(
				Provide(func() (*Foo, error) { return nil, provideTestError }),
				Invoke(func(foo *Foo) {
					if foo == nil {
						panic("foo must not be nil")
					}
				}),
			),
			error:               provideTestError,
			invokeMustBeSkipped: true,
		},
		{
			name: "invoke error",
			option: Options(
				Provide(func() *Foo { return &Foo{} }),
				Invoke(func(foo *Foo) error {
					if foo == nil {
						panic("foo must not be nil")
					}
					return invokeTestError
				}),
			),
			error:               invokeTestError,
			invokeMustBeSkipped: true,
		},
		{
			name: "provide with dependency",
			option: Options(
				Provide(
					func(foo Foo) *Foo { return &foo },
					func(bar *Bar) Foo {
						if bar == nil {
							panic("bar must not be nil")
						}
						return Foo{}
					},
					func() *Bar { return &Bar{} },
				),
				Invoke(func(foo *Foo) {
					if foo == nil {
						panic("foo must not be nil")
					}
				}),
			),
			error: nil,
		},
		{
			name: "dry run provide with dependency",
			option: Options(
				WithDryRun(),
				Provide(
					func(foo Foo) *Foo { return &foo },
					func(bar *Bar) Foo {
						if bar == nil {
							panic("bar must not be nil")
						}
						return Foo{}
					},
					func() *Bar { return &Bar{} },
				),
				Invoke(func(foo *Foo) {
					if foo == nil {
						panic("foo must not be nil")
					}
				}),
			),
			error:               nil,
			invokeMustBeSkipped: true,
		},
		{
			name: "similar package names",
			option: Options(
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
			error: nil,
		},
		{
			name: "dry run similar package names",
			option: Options(
				WithDryRun(),
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
			error:               nil,
			invokeMustBeSkipped: true,
		},
		{
			name: "multiple provide",
			option: Options(
				Provide(
					func() *Foo { return &Foo{} },
					func() *Foo { return &Foo{} },
				),
				Invoke(func(foo *Foo) {
					if foo == nil {
						panic("foo must not be nil")
					}
				}),
			),
			error: ErrMultipleProvide,
		},
		{
			name: "dry run multiple provide",
			option: Options(
				WithDryRun(),
				Provide(
					func() *Foo { return &Foo{} },
					func() *Foo { return &Foo{} },
				),
				Invoke(func(foo *Foo) {
					if foo == nil {
						panic("foo must not be nil")
					}
				}),
			),
			error: ErrMultipleProvide,
		},
		{
			name: "supply",
			option: Options(
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
		{
			name: "supply context",
			option: Options(
				WithDuckTyping(),
				Supply(context.Background()),
				Invoke(func(ctx context.Context) {
					if ctx == nil {
						panic("ctx must not be nil")
					}
				}, func(ctx context.Context) {
					if ctx == nil {
						panic("ctx must not be nil")
					}
				}),
			),
		},
		{
			name: "duck typing",
			option: Options(
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
		{
			name: "dry run duck typing",
			option: Options(
				WithDryRun(),
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
			invokeMustBeSkipped: true,
		},
		{
			name: "duck typing multiple provide",
			option: Options(
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
			error:               ErrMultipleProvide,
			invokeMustBeSkipped: true,
		},
		{
			name: "dry run duck typing multiple provide",
			option: Options(
				WithDryRun(),
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
			error:               ErrMultipleProvide,
			invokeMustBeSkipped: true,
		},
		{
			name: "cyclic_provide",
			option: Options(
				Provide(
					func(*Foo) *Bar {
						return &Bar{}
					},
					func(*Bar) *Buzz {
						return &Buzz{}
					},
					func(*Buzz) *Foo {
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
			error:               ErrCyclicProvideDetected,
			invokeMustBeSkipped: true,
		},
		{
			name: "dry run cyclic_provide",
			option: Options(
				WithDryRun(),
				Provide(
					func(*Foo) *Bar {
						return &Bar{}
					},
					func(*Bar) *Buzz {
						return &Buzz{}
					},
					func(*Buzz) *Foo {
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
			error:               ErrCyclicProvideDetected,
			invokeMustBeSkipped: true,
		},
	}

	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			called := false

			err := Revolve(ctx, testCase.option, Invoke(func() {
				called = true
			}))
			if err != testCase.error {
				if !(err == nil || testCase.error == nil) {
					if errors.Is(err, testCase.error) {
						return
					}
				}
				t.Fatalf("errors are not equal: \ngot: %v \nexp: %v", err, testCase.error)
			}
			if called == testCase.invokeMustBeSkipped {
				t.Fatal("invoke func must be called")
			}
		})
	}
}

type Foo struct{}

func (Foo) foo() {}

type Bar struct{}

func (Bar) bar() {}

type Buzz struct{}

func (Buzz) buzz() {}

type IFoo interface {
	foo()
}

type IBar interface {
	bar()
}

type FooBar struct{}

func (FooBar) foo() {}
func (FooBar) bar() {}

func customLogFunc(lvl LogLevel, format string, args ...any) {
	switch lvl {
	case LogLevelInfo:
		log.Printf("customLogFunc: "+format, args...)
	case LogLevelDebug:
		log.Printf("customLogFunc: debug:"+format, args...)
	}
}

type customLogger struct{}

func (l customLogger) Printf(lvl LogLevel, format string, args ...any) {
	switch lvl {
	case LogLevelInfo:
		log.Printf("customLogger: "+format, args...)
	case LogLevelDebug:
		log.Printf("customLogger: debug:"+format, args...)
	}
}

var provideTestError = errors.New("provide test err")
var invokeTestError = errors.New("invoke test err")
