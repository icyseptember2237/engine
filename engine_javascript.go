package engine

import (
	"github.com/robertkrimen/otto"
	"reflect"
)

const (
	TypeEngineJs = "js"
)

type JsEngine struct {
	vm    *otto.Otto
	ready bool
}

func (e *JsEngine) New() {
	e.vm = otto.New()
	e.ready = false
}

func (e *JsEngine) IsReady() bool {
	return e.ready
}

func (e *JsEngine) SetReady() {
	e.ready = true
}

func (e *JsEngine) ParseString(source string) error {
	_, err := e.vm.Run(source)
	if err != nil {
		return err
	}
	return nil
}

func (e *JsEngine) ParseFile(path string) error {
	script, err := e.vm.Compile(path, nil)
	if err != nil {
		return err
	}
	_, err = e.vm.Run(script)
	return err
}

func (e *JsEngine) RegisterObject(objectName string, objectPtr interface{}) {
	e.vm.Set(objectName, objectPtr)
}

func (e *JsEngine) RegisterFunction(goFuncName string, goFuncPtr interface{}) {
	goFuncVal := reflect.ValueOf(goFuncPtr)
	if goFuncVal.Kind() != reflect.Func {
		panic("register not invalid function")
	}

	goParamsNum := goFuncVal.Type().NumIn()
	in := make([]reflect.Value, goParamsNum)

	var fn = func(call otto.FunctionCall) otto.Value {
		for i := 0; i < goParamsNum; i++ {
			jsParam := call.Argument(i)
			val, err := jsParam.Export()
			if err != nil {
				panic(err)
			}

			in[i] = reflect.ValueOf(val)
		}
		var result otto.Value
		goRets := goFuncVal.Call(in)
		if len(goRets) > 0 {
			data := goRets[0].Interface()
			result, _ = e.vm.ToValue(data)
		} else {
			result = otto.NullValue()
		}
		return result
	}
	e.vm.Set(goFuncName, fn)
}

func (e *JsEngine) RegisterModule(moduleName string, moduleFuncPtr map[string]interface{}) {
	// not supported
}

func (e *JsEngine) IsFunction(scriptFuncName string) bool {
	// not supported
	return false
}

func (e *JsEngine) Call(scriptFuncName string, retNum int, args ...interface{}) (error, []interface{}) {
	value, err := e.vm.Call(scriptFuncName, nil, args...)
	if err != nil {
		return err, nil
	}
	data, err := value.Export()
	if err != nil {
		return err, nil
	}
	return nil, []interface{}{data}
}

func (e *JsEngine) Close() {
}
