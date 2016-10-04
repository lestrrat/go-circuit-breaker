package breaker

// NewMap creates a default breaker map
func NewMap() Map {
	return &simpleMap{
		breakers: make(map[string]Breaker),
	}
}

func (m *simpleMap) Set(name string, cb Breaker) {
	m.mutex.Lock()
	m.breakers[name] = cb
	m.mutex.Unlock()
}

func (m *simpleMap) Get(name string) (Breaker, bool) {
	m.mutex.RLock()
	cb, ok := m.breakers[name]
	m.mutex.RUnlock()

	return cb, ok
}
