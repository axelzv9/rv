package rv

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

type function struct {
	fn    *reflect.Value // maybe nil when values are supply
	layer int

	inTypes    []reflect.Type
	inProvides []*function // refs for the arguments provide functions

	outTypes  []reflect.Type
	outValues []reflect.Value
}

func (f *function) call(ctx context.Context, rv *revolver, args []reflect.Value) error {
	if f.fn == nil {
		return nil
	}

	result := make(chan []reflect.Value)
	var ts int64

	go func() {
		start := time.Now()
		values := f.fn.Call(args)
		sinceStart := time.Since(start)
		atomic.StoreInt64(&ts, int64(sinceStart))
		result <- values
	}()

	var values []reflect.Value
	select {
	case <-ctx.Done():
		return ctx.Err()
	case values = <-result:
	}

	spent := time.Duration(atomic.LoadInt64(&ts))

	if len(f.outTypes) == 0 {
		rv.printf("invoking %s completed in %s", f.String(), spent.String())
	} else {
		rv.printf("providing %s completed in %s", f.String(), spent.String())
	}

	for _, v := range values {
		if isErrorType(v.Type()) {
			err, _ := v.Interface().(error)
			if err != nil {
				return err
			}
			continue
		}
		f.outValues = append(f.outValues, v)
	}

	return nil
}

func (f *function) String() string {
	if f == nil {
		return "function is nil"
	}

	name := "noname"
	if f.fn != nil {
		name = funcName(*f.fn)
	}

	var ins, outs []string
	for _, t := range f.inTypes {
		ins = append(ins, t.String())
	}
	for _, t := range f.outTypes {
		outs = append(outs, t.String())
	}

	return fmt.Sprintf("%s(%s) (%s)", name, strings.Join(ins, ", "), strings.Join(outs, ", "))
}

func (f *function) Debug() string {
	if f == nil {
		return "function is nil"
	}

	name := "noname"
	if f.fn != nil {
		name = funcName(*f.fn)
	}

	var ins, outs []string
	for _, t := range f.inTypes {
		ins = append(ins, t.String())
	}
	for _, t := range f.outTypes {
		outs = append(outs, t.String())
	}

	var provides strings.Builder
	for _, provide := range f.inProvides {
		if provide == nil {
			provides.WriteString("null")
			continue
		}
		if provide.fn != nil {
			provides.WriteRune('\n')
			provides.WriteString(funcName(*provide.fn))
		} else {
			provides.WriteRune('\n')
			provides.WriteString(funcName(provide.outValues[0]))
		}
	}

	return fmt.Sprintf("%s(%s) (%s) layer=%d provides=[%s]",
		name, strings.Join(ins, ", "), strings.Join(outs, ", "), f.layer, provides.String())
}

func parseSupply(value any) *function {
	val := reflect.ValueOf(value)
	return &function{
		outTypes:  []reflect.Type{val.Type()},
		outValues: []reflect.Value{val},
	}
}

func parseProvide(target any) (*function, error) {
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Func {
		return nil, fmt.Errorf("%w for %s", ErrUnsupportedProvideTarget, value.Type().String())
	}

	typ := value.Type()
	var inTypes []reflect.Type
	var outTypes []reflect.Type
	for i := 0; i < typ.NumIn(); i++ {
		inTypes = append(inTypes, typ.In(i))
	}
	for i := 0; i < typ.NumOut(); i++ {
		outTypes = append(outTypes, typ.Out(i))
	}

	return &function{
		fn:         &value,
		inTypes:    inTypes,
		inProvides: make([]*function, len(inTypes)),
		outTypes:   outTypes,
	}, nil
}

func parseInvoke(target any) (*function, error) {
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Func {
		return nil, fmt.Errorf("%w for %s", ErrUnsupportedInvokeTarget, value.Type().String())
	}

	typ := value.Type()
	var inTypes []reflect.Type
	for i := 0; i < typ.NumIn(); i++ {
		inTypes = append(inTypes, typ.In(i))
	}

	return &function{
		fn:         &value,
		inTypes:    inTypes,
		inProvides: make([]*function, len(inTypes)),
	}, nil
}

func funcName(fn reflect.Value) string {
	if fn.Kind() != reflect.Func {
		return fn.String()
	}
	name := runtime.FuncForPC(fn.Pointer()).Name()
	if unescaped, err := url.QueryUnescape(name); err == nil {
		name = unescaped
	}
	return name
}
