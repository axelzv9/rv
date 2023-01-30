package rv

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

type functionState int

const (
	StateInitialized functionState = iota + 1
	StateLinked
	StateCalled
)

type function struct {
	targetFunc reflect.Value // maybe empty when values are provided by Supply
	inputs     []input
	outputs    []output
	state      functionState
}

type input struct {
	typ         reflect.Type
	provider    *function
	outputIndex int
}

type output struct {
	typ   reflect.Type
	value reflect.Value
}

func (f *function) LinkProvides(provides []*function, assignable typesAssignableFunc) (providers []*function, _ error) {
	providers = make([]*function, 0, len(f.inputs))
	for inIndex, in := range f.inputs {
		provider, outputIndex, err := f.linkInput(in.typ, provides, assignable)
		if err != nil {
			return nil, err
		}
		if provider == nil {
			return nil, fmt.Errorf("linking: %w type=%s for func %s", ErrCannotProvideValue, in.typ, f.String())
		}
		f.inputs[inIndex].provider = provider
		f.inputs[inIndex].outputIndex = outputIndex
		providers = append(providers, provider)
	}
	f.state = StateLinked
	return
}

func (f *function) State() functionState {
	return f.state
}

func (f *function) Call(ctx context.Context, logger Logger, dryRun bool) error {
	if f.state >= StateCalled {
		return nil
	}
	defer func() {
		f.state = StateCalled
	}()

	args, err := f.collectArgsValues()
	if err != nil {
		return err
	}

	if dryRun {
		return nil
	}

	result := make(chan []reflect.Value)
	var ts int64

	go func() {
		start := time.Now()
		values := f.targetFunc.Call(args)
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
	logger.Printf(LogLevelInfo, "executing %s completed in %s", f.String(), spent.String())

	for i, v := range values {
		if isErrorType(v.Type()) {
			err, _ := v.Interface().(error)
			if err != nil {
				return err
			}
			continue
		}
		f.outputs[i].value = v
	}

	return nil
}

func (f *function) linkInput(typ reflect.Type, provides []*function, assignable typesAssignableFunc) (
	provider *function, outputIndex int, err error) {
	for _, provide := range provides {
		if f == provide { // exclude self-providing
			continue
		}
		for outIndex, out := range provide.outputs {
			if isErrorType(out.typ) { // exclude providing type `error`
				continue
			}
			if !assignable(out.typ, typ) {
				continue
			}
			if provider != nil {
				return nil, 0,
					fmt.Errorf("linking: %w of type=%s \nfirst usage:  %s \nsecond usage: %s",
						ErrMultipleProvide, typ, provider.String(), provide.String(),
					)
			}
			provider = provide
			outputIndex = outIndex
		}
	}
	return
}

func (f *function) collectArgsValues() ([]reflect.Value, error) {
	var result = make([]reflect.Value, 0, len(f.inputs))
	for i := range f.inputs {
		in := f.inputs[i]
		if in.provider.State() < StateCalled {
			return nil, fmt.Errorf("%w %s", ErrCyclicProvideDetected, f.String())
		}
		if len(in.provider.outputs) <= in.outputIndex {
			return nil, fmt.Errorf("%w: failed to collect arguments for %s func: %s",
				ErrInternalError, in.typ, f.String(),
			)
		}
		result = append(result, in.provider.outputs[in.outputIndex].value)
	}
	return result, nil
}

func (f *function) String() string {
	if f == nil {
		return "function is nil"
	}

	name := funcName(f.targetFunc)
	defer func() {
		if err := recover(); err != nil {
			log.Printf("recovered: %v funcName: %s", err, name)
		}
	}()

	var ins, outs []string
	for _, in := range f.inputs {
		ins = append(ins, in.typ.String())
	}
	for _, out := range f.outputs {
		outs = append(outs, out.typ.String())
	}

	return fmt.Sprintf("%s(%s) (%s)", name, strings.Join(ins, ", "), strings.Join(outs, ", "))
}

func (f *function) Debug() string {
	if f == nil {
		return "function is nil"
	}

	name := funcName(f.targetFunc)

	var ins, outs []string
	var providers strings.Builder
	for _, in := range f.inputs {
		ins = append(ins, in.typ.String())
		if in.provider == nil {
			providers.WriteString("null")
			continue
		}
		providers.WriteRune('\n')
		providers.WriteString(funcName(in.provider.targetFunc))
	}
	for _, out := range f.outputs {
		outs = append(outs, out.typ.String())
	}

	return fmt.Sprintf("%s(%s) (%s) state=%d provides=[%s]",
		name, strings.Join(ins, ", "), strings.Join(outs, ", "), f.state, providers.String())
}

func parseSupply(value any) *function {
	val := reflect.ValueOf(value)
	return &function{
		outputs: []output{{
			typ:   val.Type(),
			value: val,
		}},
		state: StateCalled,
	}
}

func parseProvide(target any) (*function, error) {
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Func {
		return nil, fmt.Errorf("%w for %s", ErrUnsupportedProvideTarget, value.Type().String())
	}

	typ := value.Type()
	inputs := make([]input, typ.NumIn())
	outputs := make([]output, typ.NumOut())
	for i := 0; i < typ.NumIn(); i++ {
		inputs[i].typ = typ.In(i)
	}
	for i := 0; i < typ.NumOut(); i++ {
		outputs[i].typ = typ.Out(i)
	}

	return &function{
		targetFunc: value,
		inputs:     inputs,
		outputs:    outputs,
		state:      StateInitialized,
	}, nil
}

func parseInvoke(target any) (*function, error) {
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Func {
		return nil, fmt.Errorf("%w for %s", ErrUnsupportedInvokeTarget, value.Type().String())
	}

	typ := value.Type()
	inputs := make([]input, typ.NumIn())
	for i := 0; i < typ.NumIn(); i++ {
		inputs[i].typ = typ.In(i)
	}

	return &function{
		targetFunc: value,
		inputs:     inputs,
		state:      StateInitialized,
	}, nil
}

var loggerType = reflect.TypeOf((*Logger)(nil)).Elem()
var logFuncType = reflect.TypeOf((*LogFunc)(nil)).Elem()

func parseLoggerProvide(target any) (*function, error) {
	value := reflect.ValueOf(target)
	typ := value.Type()
	kind := value.Kind()
	switch {
	case kind == reflect.Func && typ.AssignableTo(logFuncType):
		return &function{
			outputs: []output{{
				typ:   logFuncType,
				value: value.Convert(logFuncType),
			}},
			state: StateCalled,
		}, nil
	case typ.AssignableTo(loggerType):
		return &function{
			outputs: []output{{
				typ:   loggerType,
				value: value,
			}},
			state: StateCalled,
		}, nil
	case kind != reflect.Func:
		return nil, fmt.Errorf("%w for %s", ErrUnsupportedLoggerProvider, typ.String())
	}

	inputs := make([]input, typ.NumIn())
	outputs := make([]output, typ.NumOut())
	for i := 0; i < typ.NumIn(); i++ {
		inputs[i].typ = typ.In(i)
	}
	for i := 0; i < typ.NumOut(); i++ {
		outputs[i].typ = typ.Out(i)
	}
	return &function{
		targetFunc: value,
		inputs:     inputs,
		outputs:    outputs,
		state:      StateInitialized,
	}, nil
}

func funcName(fn reflect.Value) string {
	if fn.Kind() != reflect.Func {
		return "noname"
	}
	name := runtime.FuncForPC(fn.Pointer()).Name()
	if unescaped, err := url.QueryUnescape(name); err == nil {
		name = unescaped
	}
	return name
}
