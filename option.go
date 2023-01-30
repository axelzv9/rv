package rv

type Option interface {
	apply(*revolver) error
}

func Options(opts ...Option) Option {
	return optionGroup(opts)
}

func Supply(values ...any) Option {
	opts := make([]Option, 0, len(values))
	for _, value := range values {
		opts = append(opts, supplyOption(value))
	}
	return Options(opts...)
}

func Provide(funcs ...any) Option {
	opts := make([]Option, 0, len(funcs))
	for _, fn := range funcs {
		opts = append(opts, provideOption(fn))
	}
	return Options(opts...)
}

func Invoke(funcs ...any) Option {
	var opts []Option
	for _, fn := range funcs {
		opts = append(opts, invokeOption(fn))
	}
	return Options(opts...)
}

func WithDuckTyping() Option {
	return optionFunc(func(rv *revolver) error {
		rv.assignable = duckTypingAssignable
		return nil
	})
}

func WithDryRun() Option {
	return optionFunc(func(rv *revolver) error {
		rv.dryRun = true
		return nil
	})
}

func WithLogger(target any) Option {
	return optionFunc(func(rv *revolver) error {
		provide, err := parseLoggerProvide(target)
		if err != nil {
			return err
		}
		invoke, _ := parseInvoke(func(logger Logger) {
			rv.logger = logger
		})
		rv.provides = append(rv.provides, provide)
		rv.loggerInvoker = invoke
		return nil
	})
}

type optionGroup []Option

func (og optionGroup) apply(rv *revolver) error {
	for _, opt := range og {
		if err := opt.apply(rv); err != nil {
			return err
		}
	}
	return nil
}

type optionFunc func(*revolver) error

func (of optionFunc) apply(rv *revolver) error {
	return of(rv)
}

func supplyOption(value any) optionFunc {
	return func(rv *revolver) error {
		rv.provides = append(rv.provides, parseSupply(value))
		return nil
	}
}

func provideOption(target any) optionFunc {
	return func(rv *revolver) error {
		provide, err := parseProvide(target)
		if err != nil {
			return err
		}
		rv.provides = append(rv.provides, provide)
		return nil
	}
}

func invokeOption(target any) optionFunc {
	return func(rv *revolver) error {
		invoke, err := parseInvoke(target)
		if err != nil {
			return err
		}
		rv.invokes = append(rv.invokes, invoke)
		return nil
	}
}

// func loggerOption(target any) optionFunc {
// switch logger := target.(type) {
// case Logger:
// 	return func(rv *revolver) error {
// 		rv.logger = logger
// 		return nil
// 	}
// case LogFunc:
// 	return func(rv *revolver) error {
// 		rv.logger = logger
// 		return nil
// 	}
// }
// return
// }
