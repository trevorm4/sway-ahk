package main

import (
	"log"
	"sync"
)

type pauseManager struct {
	mu        sync.Mutex
	appClass  string
	pauseKey  int
	appScoped bool
	global    bool
	apps      map[string]bool
}

var pauseState = &pauseManager{apps: make(map[string]bool)}

func (m *pauseManager) setActive(appClass string, pauseKey int, appScoped bool) {
	m.mu.Lock()
	m.appClass = appClass
	m.pauseKey = pauseKey
	m.appScoped = appScoped
	m.mu.Unlock()
}

func (m *pauseManager) toggle(code int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.pauseKey == 0 || code != m.pauseKey {
		return false
	}
	if m.appScoped {
		m.apps[m.appClass] = !m.apps[m.appClass]
		logPause(m.apps[m.appClass], "for "+m.appClass)
	} else {
		m.global = !m.global
		logPause(m.global, "globally")
	}
	return true
}

func (m *pauseManager) paused() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.global || m.apps[m.appClass]
}

func logPause(paused bool, scope string) {
	if paused {
		log.Printf("paused %s", scope)
	} else {
		log.Printf("resumed %s", scope)
	}
}
