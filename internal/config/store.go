package config

import (
	"sync"
	"sync/atomic"
)

// Watcher is a function that gets called when configuration changes
type Watcher func(newCfg *Config, changed map[string]bool)

// Store provides thread-safe access to configuration with watchers and validators
type Store struct {
	v          atomic.Value // *Config
	mu         sync.RWMutex
	watchers   []Watcher
	validators []Validator
}

// NewStore creates a new configuration store
func NewStore(cfg *Config) *Store {
	s := &Store{}
	s.v.Store(cfg)
	return s
}

// Get returns the current configuration
func (s *Store) Get() *Config {
	return s.v.Load().(*Config)
}

// Update sets a new configuration and notifies all watchers
func (s *Store) Update(newCfg *Config, changed map[string]bool) {
	s.v.Store(newCfg)
	s.mu.RLock()
	ws := append([]Watcher(nil), s.watchers...)
	s.mu.RUnlock()
	for _, w := range ws {
		w(newCfg, changed)
	}
}

// Watch registers a watcher for configuration changes and returns an unwatch function
func (s *Store) Watch(w Watcher) func() {
	s.mu.Lock()
	s.watchers = append(s.watchers, w)
	idx := len(s.watchers) - 1
	s.mu.Unlock()
	return func() {
		s.mu.Lock()
		// remove index
		if idx >= 0 && idx < len(s.watchers) {
			s.watchers = append(s.watchers[:idx], s.watchers[idx+1:]...)
		}
		s.mu.Unlock()
	}
}

// Validator is a function that validates configuration changes
type Validator func(newCfg *Config, changed map[string]bool) error

// AddValidator registers a validator. If any validator returns error on update, the update will be discarded.
func (s *Store) AddValidator(v Validator) func() {
	s.mu.Lock()
	s.validators = append(s.validators, v)
	idx := len(s.validators) - 1
	s.mu.Unlock()
	return func() {
		s.mu.Lock()
		if idx >= 0 && idx < len(s.validators) {
			s.validators = append(s.validators[:idx], s.validators[idx+1:]...)
		}
		s.mu.Unlock()
	}
}

// UpdateValidated runs validators before committing the config. If any validator fails, no change is applied.
func (s *Store) UpdateValidated(newCfg *Config, changed map[string]bool) bool {
	s.mu.RLock()
	vals := append([]Validator(nil), s.validators...)
	s.mu.RUnlock()
	for _, v := range vals {
		if err := v(newCfg, changed); err != nil {
			return false
		}
	}
	s.Update(newCfg, changed)
	return true
}

func cloneConfig(in *Config) *Config {
	out := *in
	return &out
}
