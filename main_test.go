package main

import (
	"testing"
	"time"
)

func TestParseKeyPressed(t *testing.T) {
	tests := []struct {
		name string
		line string
		code int
		ok   bool
	}{
		{"press", "-event18  KEYBOARD_KEY     +2.36s  KEY_E (18) pressed", 18, true},
		{"release ignored", "-event18  KEYBOARD_KEY     +2.42s  KEY_E (18) released", 0, false},
		{"pointer button ignored", "-event6   POINTER_BUTTON    +1.20s  BTN_LEFT (272) pressed", 0, false},
		{"device added ignored", "-event6   DEVICE_ADDED      Power Button      seat0 default cap:d", 0, false},
		{"no keycode", "-event18  KEYBOARD_KEY     +2.36s  pressed", 0, false},
		{"garbage", "not an event line", 0, false},
		{"empty", "", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, ok := parseKeyPressed(tt.line)
			if ok != tt.ok || code != tt.code {
				t.Errorf("parseKeyPressed(%q) = (%d, %v), want (%d, %v)", tt.line, code, ok, tt.code, tt.ok)
			}
		})
	}
}

func TestRemapManager(t *testing.T) {
	m := &remapManager{recent: make(map[int]time.Time)}
	m.setRemaps([]RemapAction{{Key: "e", To: []string{"q", "w", "e"}}})

	targets, delay, ok := m.targetsFor(keyCodeMap["e"])
	if !ok {
		t.Fatal("expected remap for e to be active")
	}
	if len(targets) != 3 || targets[0] != keyCodeMap["q"] || targets[1] != keyCodeMap["w"] || targets[2] != keyCodeMap["e"] {
		t.Errorf("unexpected targets: %v", targets)
	}
	if delay != defaultRemapDelayMs {
		t.Errorf("delay = %d, want %d", delay, defaultRemapDelayMs)
	}

	m.markInjected(keyCodeMap["q"])
	if _, _, ok := m.targetsFor(keyCodeMap["q"]); ok {
		t.Error("recently injected key should not trigger a remap")
	}

	m.recent[keyCodeMap["q"]] = time.Now().Add(-2 * injectedKeyGrace)
	if _, _, ok := m.targetsFor(keyCodeMap["q"]); ok {
		t.Error("key with expired injection grace should not trigger a remap")
	}

	m.setRemaps(nil)
	if _, _, ok := m.targetsFor(keyCodeMap["e"]); ok {
		t.Error("no remaps should be active after clearing")
	}
}

func TestSetRemapsSkipsUnknownKeys(t *testing.T) {
	m := &remapManager{recent: make(map[int]time.Time)}
	m.setRemaps([]RemapAction{{Key: "e", To: []string{"q", "nope"}}})

	targets, _, ok := m.targetsFor(keyCodeMap["e"])
	if !ok {
		t.Fatal("expected remap for e to be active")
	}
	if len(targets) != 1 || targets[0] != keyCodeMap["q"] {
		t.Errorf("unexpected targets: %v", targets)
	}
}
