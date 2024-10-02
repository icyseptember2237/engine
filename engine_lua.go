package engine

import (
	"crypto/tls"
	"fmt"
	"github.com/ailncode/gluaxmlpath"
	"github.com/ciaos/gluahttp"
	"github.com/cjoudrey/gluaurl"
	"github.com/yuin/gluamapper"
	"github.com/yuin/gluare"
	lua "github.com/yuin/gopher-lua"
	luajson "layeh.com/gopher-json"
	luar "layeh.com/gopher-luar"
	"net/http"
	"reflect"
	"unicode"
)

const (
	TypeEngineLua = "lua"
)

type LuaEngine struct {
	vm    *lua.LState
	ready bool
}

func (e *LuaEngine) New() {
	e.vm = lua.NewState()
	luajson.Preload(e.vm)
	e.vm.PreloadModule("url", gluaurl.Loader)
	e.vm.PreloadModule("re", gluare.Loader)
	e.vm.PreloadModule("http", gluahttp.NewHttpModule(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}).Loader)
	e.vm.PreloadModule("xmlpath", gluaxmlpath.Loader)
	e.ready = false
}

func (e *LuaEngine) IsReady() bool {
	return e.ready
}
func (e *LuaEngine) SetReady() {
	e.ready = true
}

func (e *LuaEngine) ParseString(source string) error {
	return e.vm.DoString(source)
}

func (e *LuaEngine) ParseFile(path string) error {
	return e.vm.DoFile(path)
}

func (e *LuaEngine) toLuaValue(src interface{}) lua.LValue {
	if src == nil {
		return lua.LNil
	}
	if reflect.ValueOf(src).Kind() == reflect.Map {
		dst := &lua.LTable{}
		srcVal := reflect.ValueOf(src)
		for _, key := range srcVal.MapKeys() {
			dst.RawSet(luar.New(e.vm, key.Interface()), e.toLuaValue(srcVal.MapIndex(key).Interface()))
		}
		return dst
	} else if reflect.ValueOf(src).Kind() == reflect.Slice {
		dst := &lua.LTable{}
		srcVal := reflect.ValueOf(src)
		for i := 0; i < srcVal.Len(); i++ {
			dst.Append(e.toLuaValue(srcVal.Index(i).Interface()))
		}
		return dst
	} else {
		return luar.New(e.vm, src)
	}
}

func (e *LuaEngine) toGoValue(src lua.LValue) interface{} {
	switch v := src.(type) {
	case *lua.LTable:
		maxn := v.MaxN()
		if maxn == 0 { // table
			ret := make(map[string]interface{})
			v.ForEach(func(key, value lua.LValue) {
				keyStr := fmt.Sprint(e.toGoValue(key))
				if unicode.IsLower(rune(keyStr[0])) {
					ret[gluamapper.ToUpperCamelCase(keyStr)] = e.toGoValue(value)
				} else {
					ret[keyStr] = e.toGoValue(value)
				}
			})
			return ret
		} else { // array
			ret := make([]interface{}, 0, maxn)
			for i := 1; i <= maxn; i++ {
				ret = append(ret, e.toGoValue(v.RawGetInt(i)))
			}
			return ret
		}
	case *lua.LUserData:
		return reflect.ValueOf(gluamapper.ToGoValue(src, gluamapper.Option{NameFunc: gluamapper.ToUpperCamelCase})).Interface().(*lua.LUserData).Value
	default:
		return gluamapper.ToGoValue(src, gluamapper.Option{NameFunc: gluamapper.ToUpperCamelCase})
	}
}

func (e *LuaEngine) RegisterObject(objectName string, objectPtr interface{}) {
	dst := e.toLuaValue(objectPtr)
	e.vm.SetGlobal(objectName, dst)
}

func (e *LuaEngine) RegisterFunction(goFuncName string, goFuncPtr interface{}) {

	goFuncVal := reflect.ValueOf(goFuncPtr)
	if goFuncVal.Kind() != reflect.Func {
		panic("register not invalid function")
	}

	goParamsNum := goFuncVal.Type().NumIn()
	goRetsNum := goFuncVal.Type().NumOut()
	in := make([]reflect.Value, goParamsNum)

	var fn = func(L *lua.LState) int {
		for i := 0; i < goParamsNum; i++ {
			luaParam := L.Get(i + 1)
			in[i] = reflect.ValueOf(e.toGoValue(luaParam))
		}

		goRet := goFuncVal.Call(in)
		for i := 0; i < len(goRet); i++ {
			L.Push(e.toLuaValue(goRet[i].Interface()))
		}

		return goRetsNum
	}
	e.vm.SetGlobal(goFuncName, e.vm.NewFunction(fn))
}

func (e *LuaEngine) RegisterModule(moduleName string, moduleFuncPtr map[string]interface{}) {
	exports := make(map[string]lua.LGFunction)
	for goFuncName, goFuncPtr := range moduleFuncPtr {
		goFuncVal := reflect.ValueOf(goFuncPtr)
		if goFuncVal.Kind() != reflect.Func {
			panic("register not invalid function")
		}

		goParamsNum := goFuncVal.Type().NumIn()
		goRetsNum := goFuncVal.Type().NumOut()

		var fn = func(L *lua.LState) int {
			in := make([]reflect.Value, goParamsNum)

			for i := 0; i < goParamsNum; i++ {
				luaParam := L.Get(i + 1)
				in[i] = reflect.ValueOf(e.toGoValue(luaParam))
			}

			goRet := goFuncVal.Call(in)
			for i := 0; i < len(goRet); i++ {
				L.Push(e.toLuaValue(goRet[i].Interface()))
			}

			return goRetsNum
		}
		exports[goFuncName] = fn
	}

	e.vm.PreloadModule(moduleName, func(L *lua.LState) int {
		// register functions to the table
		mod := L.SetFuncs(L.NewTable(), exports)
		// register other stuff
		L.SetField(mod, "name", lua.LString(moduleName))

		// returns the module
		L.Push(mod)
		return 1
	})
}

func (e *LuaEngine) IsFunction(scriptFuncName string) bool {
	val := e.vm.GetGlobal(scriptFuncName)
	if val.Type() == lua.LTFunction {
		return true
	}
	return false
}

func (e *LuaEngine) Call(scriptFuncName string, retNum int, args ...interface{}) (error, []interface{}) {

	luaArgs := make([]lua.LValue, len(args))
	for i := 0; i < len(args); i++ {
		luaArgs[i] = e.toLuaValue(args[i])
	}

	if err := e.vm.CallByParam(lua.P{
		Fn:      e.vm.GetGlobal(scriptFuncName),
		NRet:    retNum,
		Protect: true,
		Handler: nil,
	}, luaArgs...); err != nil {
		return err, nil
	}

	rets := make([]interface{}, 0)
	for i := 0; i < retNum; i++ {
		luaRet := e.vm.Get(-1)
		e.vm.Pop(1)

		res := e.toGoValue(luaRet)
		rets = append([]interface{}{res}, rets...)
	}

	return nil, rets
}

func (e *LuaEngine) Close() {
	e.vm.Close()
}

func (e *LuaEngine) GetVM() *lua.LState {
	return e.vm
}
