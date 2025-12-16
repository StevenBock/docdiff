package language

import "sync"

type Registry struct {
	mu         sync.RWMutex
	strategies map[string]Strategy
	extMap     map[string]Strategy
}

func NewRegistry() *Registry {
	return &Registry{
		strategies: make(map[string]Strategy),
		extMap:     make(map[string]Strategy),
	}
}

func DefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register(NewPHPStrategy())
	r.Register(NewGoStrategy())
	r.Register(NewJavaStrategy())
	r.Register(NewPythonStrategy())
	r.Register(NewJavaScriptStrategy())
	r.Register(NewRubyStrategy())
	return r
}

func (r *Registry) Register(s Strategy) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.strategies[s.Name()] = s
	for _, ext := range s.Extensions() {
		r.extMap[ext] = s
	}
}

func (r *Registry) GetByExtension(ext string) (Strategy, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.extMap[ext]
	return s, ok
}

func (r *Registry) GetByName(name string) (Strategy, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.strategies[name]
	return s, ok
}

func (r *Registry) AllStrategies() []Strategy {
	r.mu.RLock()
	defer r.mu.RUnlock()
	strategies := make([]Strategy, 0, len(r.strategies))
	for _, s := range r.strategies {
		strategies = append(strategies, s)
	}
	return strategies
}

func (r *Registry) AllExtensions() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	exts := make([]string, 0, len(r.extMap))
	for ext := range r.extMap {
		exts = append(exts, ext)
	}
	return exts
}
