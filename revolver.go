package rv

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

var (
	ErrUnsupportedProvideTarget  = errors.New("unsupported provide target")
	ErrUnsupportedLoggerProvider = errors.New("unsupported logger provider")
	ErrUnsupportedInvokeTarget   = errors.New("unsupported invoke target")
	ErrMultipleProvide           = errors.New("multiple provide")
	ErrCannotProvideValue        = errors.New("cannot provide value")
	ErrCyclicProvideDetected     = errors.New("cyclic provide detected")
	ErrInternalError             = errors.New("internal error")
)

func Revolve(ctx context.Context, opts ...Option) error {
	rv := &revolver{
		logger:     LogFunc(devNull),
		assignable: typesSimpleAssignable,
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt.apply(rv); err != nil {
			return err
		}
	}

	if err := rv.resolveLogger(ctx); err != nil {
		return err
	}

	rv.logger.Printf(LogLevelInfo, "all options have been applied")

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return rv.resolve(ctx)
}

type revolver struct {
	logger        Logger
	loggerInvoker *function
	assignable    typesAssignableFunc
	dryRun        bool

	provides []*function // provide functions instances
	invokes  []*function // invoke functions instances
}

func (rv *revolver) resolve(ctx context.Context) error {
	if rv.dryRun {
		rv.logger.Printf(LogLevelInfo, "dry run mode")
	}

	for _, p := range rv.provides {
		rv.logger.Printf(LogLevelInfo, "provide %s", p.String())
	}

	for _, fn := range rv.invokes {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		provides, err := fn.LinkProvides(rv.provides, rv.assignable)
		if err != nil {
			return err
		}
		err = rv.dfs(ctx, provides, rv.assignable, 1)
		if err != nil {
			return err
		}
	}

	rv.logger.Printf(LogLevelInfo, "all provides have been linked")

	for _, fn := range rv.invokes {
		err := fn.Call(ctx, rv.logger, rv.dryRun)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rv *revolver) dfs(ctx context.Context, funcs []*function, assignable typesAssignableFunc, depth int) error {
	for _, fn := range funcs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if fn.State() == StateInitialized {
			rv.logger.Printf(LogLevelDebug, "[%d] link provides: %s ", depth, fn.Debug())
			providers, err := fn.LinkProvides(rv.provides, assignable)
			if err != nil {
				return err
			}
			err = rv.dfs(ctx, providers, assignable, depth+1)
			if err != nil {
				if errors.Is(err, ErrCyclicProvideDetected) {
					err = fmt.Errorf("%w -> %s", err, fn.String())
				}
				return err
			}
		}
		rv.logger.Printf(LogLevelDebug, "[%d] call: %s", depth, fn.Debug())
		if err := fn.Call(ctx, rv.logger, rv.dryRun); err != nil {
			return err
		}
	}
	return nil
}

func (rv *revolver) resolveLogger(ctx context.Context) error {
	if rv.loggerInvoker == nil {
		return nil
	}
	return rv.dfs(ctx, []*function{rv.loggerInvoker}, duckTypingAssignable, 1)
}

type typesAssignableFunc func(t1, t2 reflect.Type) bool

func typesSimpleAssignable(t1, t2 reflect.Type) bool {
	return t1 == t2
}

func duckTypingAssignable(t1, t2 reflect.Type) bool {
	return t1 == t2 || t1.AssignableTo(t2) || t2.AssignableTo(t1)
}

func isErrorType(v reflect.Type) bool {
	return v.Kind() == reflect.Interface && v.String() == "error"
}
