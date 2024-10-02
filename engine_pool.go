package engine

import "sync"

type EnginePool struct {
	engineType string
	m          sync.Mutex
	saved      []Engine
}

func (ep *EnginePool) Get() Engine {
	ep.m.Lock()
	defer ep.m.Unlock()
	n := len(ep.saved)
	if n == 0 {
		return ep.New()
	}
	x := ep.saved[n-1]
	ep.saved = ep.saved[0 : n-1]
	return x
}

func (ep *EnginePool) Put(e Engine) {
	ep.m.Lock()
	defer ep.m.Unlock()
	ep.saved = append(ep.saved, e)
}

func (ep *EnginePool) Shutdown() {
	for _, e := range ep.saved {
		e.Close()
	}
}

func (ep *EnginePool) New() Engine {
	var engine Engine
	if ep.engineType == TypeEngineLua {
		engine = &LuaEngine{}
	} else if ep.engineType == TypeEngineJs {
		engine = &JsEngine{}
	} else if ep.engineType == TypeEngineGo {
		engine = &GoEngine{}
	}

	engine.New()
	return engine
}

func InitEnginePool(engineType string) *EnginePool {
	return &EnginePool{
		engineType: engineType,
		m:          sync.Mutex{},
		saved:      make([]Engine, 0, 4),
	}
}
