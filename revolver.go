package rv

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
)

var (
	ErrUnsupportedProvideTarget = errors.New("unsupported provide target")
	ErrUnsupportedInvokeTarget  = errors.New("unsupported invoke target")
	ErrMultipleProvide          = errors.New("multiple provide")
	ErrCannotProvideValue       = errors.New("cannot provide value")
	ErrCyclicProvideDetected    = errors.New("cyclic provide detected")
)

func Revolve(ctx context.Context, opts ...Option) error {
	rv := &revolver{printf: devNull, debugf: devNull}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt.apply(rv); err != nil {
			return err
		}
	}

	rv.printf("all options have been applied")

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return rv.resolve(ctx)
}

type revolver struct {
	printf     LogFunc
	debugf     LogFunc
	dryRun     bool
	duckTyping bool

	provides []*function // provide functions instances
	invokes  []*function // invoke functions instances
}

func (rv *revolver) resolve(ctx context.Context) error {
	for _, p := range rv.provides {
		rv.printf("provide %s", p.String())
	}

	assignable := typesSimpleAssignable
	if rv.duckTyping {
		rv.printf("duck typing enabled")
		assignable = duckTypingAssignable
	}

	funcs := append(rv.invokes, rv.provides...)

	for _, fn := range funcs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := linkProvides(fn, rv.provides, assignable); err != nil {
			return err
		}
	}

	rv.printf("all provides have been linked")

	if rv.dryRun {
		return nil
	}

	err := setLayers(rv.invokes, 1)
	if err != nil {
		return err
	}

	// sort funcs to call in right order
	sort.Slice(funcs, func(i, j int) bool {
		return funcs[i].layer > funcs[j].layer
	})

	for _, fn := range funcs {
		rv.debugf("%s", fn.Debug())
	}

	for _, fn := range funcs {
		if fn.layer == 0 { // ignore useless funcs
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		argsValues, err := collectArgsValues(fn, assignable)
		if err != nil {
			return err
		}
		err = fn.call(ctx, rv, argsValues)
		if err != nil {
			return err
		}
	}

	return nil
}

func linkProvides(fn *function, provides []*function, assignable typesAssignableFunc) error {
	return reflectTypes(fn.inTypes).forEach(func(indexIn int, inType reflect.Type) error {
		var desired *function
		for _, provide := range provides {
			if fn == provide { // exclude self-providing
				continue
			}
			err := reflectTypes(provide.outTypes).forEach(func(indexOut int, outType reflect.Type) error {
				if isErrorType(outType) { // exclude providing type `error`
					return nil
				}
				if !assignable(outType, inType) {
					return nil // continue looking for assignable out type
				}
				if desired != nil {
					return fmt.Errorf("linking: %w of type=%s \nfirst usage: %s \nsecond usage:%s",
						ErrMultipleProvide, inType, desired.String(), provide.String(),
					)
				}
				desired = provide
				return nil // continue searching for a matching out type, wish to find multiple providing
			})
			if err != nil {
				return err
			}
		}
		if desired == nil {
			return fmt.Errorf("linking: %w type=%s for func %s", ErrCannotProvideValue, inType, fn.String())
		}
		fn.inProvides[indexIn] = desired
		return nil
	})
}

func setLayers(funcs []*function, layer int) error {
	if layer > 1000 {
		return ErrCyclicProvideDetected
	}
	for _, fn := range funcs {
		if fn.layer < layer {
			fn.layer = layer
		}
		if err := setLayers(fn.inProvides, layer+1); err != nil {
			return err
		}
	}
	return nil
}

func collectArgsValues(fn *function, assignable typesAssignableFunc) ([]reflect.Value, error) {
	var result = make([]reflect.Value, 0, len(fn.inTypes))
	err := reflectTypes(fn.inTypes).forEach(func(index int, inType reflect.Type) error {
		provide := fn.inProvides[index]
		for _, value := range provide.outValues {
			if assignable(value.Type(), inType) {
				result = append(result, value)
				return nil // for continue
			}
		}
		return fmt.Errorf("internal error: collecting arguments error for %s func: %s", inType, fn.String())
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

type typesAssignableFunc func(t1, t2 reflect.Type) bool

func typesSimpleAssignable(t1, t2 reflect.Type) bool {
	return t1 == t2
}

func duckTypingAssignable(t1, t2 reflect.Type) bool {
	return t1 == t2 || t1.AssignableTo(t2) || t2.AssignableTo(t1)
}

type reflectTypes []reflect.Type

func (rt reflectTypes) forEach(fn func(index int, typ reflect.Type) error) error {
	for i, t := range rt {
		if err := fn(i, t); err != nil {
			return err
		}
	}
	return nil
}

func isErrorType(v reflect.Type) bool {
	return v.Kind() == reflect.Interface && v.String() == "error"
}
