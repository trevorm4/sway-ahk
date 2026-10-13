package main

import "testing"

func TestPauseGlobal(t *testing.T) {
	m := &pauseManager{apps: make(map[string]bool)}
	m.setActive("anyapp", keyCodeMap["p"], false)

	if m.paused() {
		t.Fatal("should not be paused initially")
	}
	if !m.toggle(keyCodeMap["p"]) {
		t.Fatal("pause key should be handled")
	}
	if !m.paused() {
		t.Fatal("should be paused after toggling global pause")
	}
	if !m.toggle(keyCodeMap["p"]) {
		t.Fatal("pause key should be handled")
	}
	if m.paused() {
		t.Fatal("should be resumed after second toggle")
	}
	if m.toggle(keyCodeMap["e"]) {
		t.Error("other keys should not toggle pause")
	}
}

func TestPauseAppScoped(t *testing.T) {
	m := &pauseManager{apps: make(map[string]bool)}
	m.setActive("game", keyCodeMap["p"], true)

	if !m.toggle(keyCodeMap["p"]) || !m.paused() {
		t.Fatal("app-scoped pause should pause the current app")
	}

	m.setActive("other", 0, false)
	if m.paused() {
		t.Fatal("app pause should not leak to other apps")
	}

	m.setActive("game", keyCodeMap["p"], true)
	if !m.paused() {
		t.Fatal("app pause should persist across focus changes")
	}
}

func TestPauseNoKeyConfigured(t *testing.T) {
	m := &pauseManager{apps: make(map[string]bool)}
	m.setActive("game", 0, false)
	if m.toggle(keyCodeMap["p"]) {
		t.Error("nothing should toggle when no pause key is configured")
	}
}
