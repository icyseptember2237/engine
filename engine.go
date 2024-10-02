package engine

type Engine interface {
	New()

	IsReady() bool
	SetReady()

	ParseString(source string) error
	ParseFile(path string) error

	RegisterObject(objectName string, objectPtr interface{})
	RegisterFunction(goFuncName string, goFuncPtr interface{})
	RegisterModule(moduleName string, moduleFuncPtr map[string]interface{})

	IsFunction(scriptFuncName string) bool
	Call(scriptFuncName string, retNum int, args ...interface{}) (error, []interface{})

	Close()
}
