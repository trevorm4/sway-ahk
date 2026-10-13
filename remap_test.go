package main

import (
	"testing"
	"time"
)

func TestParseKeyEvent(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		code    int
		pressed bool
		device  string
		ok      bool
	}{
		{"press", "-event18  KEYBOARD_KEY     +2.36s  KEY_E (18) pressed", 18, true, "event18", true},
		{"release", "event19  KEYBOARD_KEY     +2.42s  KEY_E (18) released", 18, false, "event19", true},
		{"pointer button ignored", "-event6   POINTER_BUTTON    +1.20s  BTN_LEFT (272) pressed", 0, false, "", false},
		{"device added ignored", "-event6   DEVICE_ADDED      Power Button      seat0 default cap:d", 0, false, "", false},
		{"no keycode", "-event18  KEYBOARD_KEY     +2.36s  pressed", 0, false, "", false},
		{"garbage", "not an event line", 0, false, "", false},
		{"empty", "", 0, false, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev, ok := parseKeyEvent(tt.line)
			if ok != tt.ok || ev.code != tt.code || ev.pressed != tt.pressed || ev.device != tt.device {
				t.Errorf("parseKeyEvent(%q) = (%+v, %v), want (code=%d, pressed=%v, device=%q, ok=%v)",
					tt.line, ev, ok, tt.code, tt.pressed, tt.device, tt.ok)
			}
		})
	}
}

func TestIsOwnEvent(t *testing.T) {
	savedTag := ownEventTag
	defer func() { ownEventTag = savedTag }()

	ownEventTag = "event19"
	if !isOwnEvent(keyEvent{device: "event19", code: 18}) {
		t.Error("events from our own device should be filtered")
	}
	if isOwnEvent(keyEvent{device: "event4", code: 18}) {
		t.Error("events from other devices should not be filtered")
	}

	ownEventTag = ""
	clearInjectedMarks()
	remapState.markInjected(keyCodeMap["w"])
	if !isOwnEvent(keyEvent{device: "event4", code: keyCodeMap["w"]}) {
		t.Error("time-based fallback should filter recently injected keys")
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
