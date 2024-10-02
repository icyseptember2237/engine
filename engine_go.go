package engine

import (
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"reflect"
)

const (
	TypeEngineGo = "go"
)

type GoEngine struct {
	i       *interp.Interpreter
	symbols map[string]reflect.Value
	fn      map[string]reflect.Value
	ready   bool
}

func (e *GoEngine) New() {
	e.i = interp.New(interp.Options{})
	e.symbols = make(map[string]reflect.Value)
	e.fn = make(map[string]reflect.Value)
	err := e.i.Use(stdlib.Symbols)
	if err != nil {
		panic(err)
	}
	e.ready = false
}

func (e *GoEngine) IsReady() bool {
	return e.ready
}
func (e *GoEngine) SetReady() {
	symbols := map[string]map[string]reflect.Value{
		"gos/gos": e.symbols,
	}
	e.i.Use(symbols)
	e.ready = true
}

func (e *GoEngine) ParseString(source string) error {
	_, err := e.i.Eval(source)
	return err
}

func (e *GoEngine) ParseFile(path string) error {
	_, err := e.i.EvalPath(path)
	return err
}

func (e *GoEngine) RegisterObject(objectName string, objectPtr interface{}) {
	e.symbols[objectName] = reflect.ValueOf(objectPtr)
}

func (e *GoEngine) RegisterFunction(goFuncName string, goFuncPtr interface{}) {
	e.symbols[goFuncName] = reflect.ValueOf(goFuncPtr)
}

func (e *GoEngine) RegisterModule(moduleName string, moduleFuncPtr map[string]interface{}) {
	modFuncSymbols := make(map[string]reflect.Value)
	for k, v := range moduleFuncPtr {
		modFuncSymbols[k] = reflect.ValueOf(v)
	}
	symbols := map[string]map[string]reflect.Value{
		moduleName: modFuncSymbols,
	}
	e.i.Use(symbols)
}

func (e *GoEngine) IsFunction(scriptFuncName string) bool {
	// not supported
	return false
}

func (e *GoEngine) Call(scriptFuncName string, retNum int, args ...interface{}) (error, []interface{}) {
	var f reflect.Value
	var ok bool
	var err error
	if f, ok = e.fn[scriptFuncName]; !ok {
		f, err = e.i.Eval(scriptFuncName)
		if err != nil {
			return err, nil
		}
		e.fn[scriptFuncName] = f
	}
	params := make([]reflect.Value, 0)
	for i := 0; i < len(args); i++ {
		params = append(params, reflect.ValueOf(args[i]))
	}
	rets := f.Call(params)

	results := make([]interface{}, 0)
	for i := 0; i < len(rets); i++ {
		results = append(results, rets[i].Interface())
	}
	return nil, results
}

func (e *GoEngine) Close() {
}
