package main

import (
	"sync"
	"testing"
	"time"
)

type clickRecorder struct {
	mu     sync.Mutex
	clicks []int
}

func (r *clickRecorder) record(code int) {
	r.mu.Lock()
	r.clicks = append(r.clicks, code)
	r.mu.Unlock()
}

func (r *clickRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.clicks)
}

func (r *clickRecorder) first() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.clicks) == 0 {
		return -1
	}
	return r.clicks[0]
}

func newTestAutoclickManager(rec *clickRecorder) *autoclickManager {
	m := newAutoclickManager()
	m.click = rec.record
	return m
}

func waitForClicks(t *testing.T, rec *clickRecorder, want int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if rec.count() >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d clicks, got %d", want, rec.count())
}

// assertClicksStable fails if the click count changes within the window.
// It first waits settle to let any in-flight tick finish.
func assertClicksStable(t *testing.T, rec *clickRecorder, settle, window time.Duration) {
	t.Helper()
	time.Sleep(settle)
	before := rec.count()
	time.Sleep(window)
	if got := rec.count(); got != before {
		t.Fatalf("expected clicking to be stopped, count changed %d -> %d", before, got)
	}
}

func TestResolveClickTrigger(t *testing.T) {
	trigger, ok := resolveClickTrigger("f8")
	if !ok || trigger.button || trigger.code != keyCodeMap["f8"] {
		t.Errorf("keyboard trigger = (%+v, %v)", trigger, ok)
	}
	trigger, ok = resolveClickTrigger("mouse_side")
	if !ok || !trigger.button || trigger.code != mouseButtonMap["side"] {
		t.Errorf("mouse trigger = (%+v, %v)", trigger, ok)
	}
	if _, ok = resolveClickTrigger("mouse_nope"); ok {
		t.Error("unknown mouse button should fail")
	}
	if _, ok = resolveClickTrigger("nope"); ok {
		t.Error("unknown key should fail")
	}
}

func TestAutoclickToggle(t *testing.T) {
	resetPauseState()
	rec := &clickRecorder{}
	m := newTestAutoclickManager(rec)
	defer m.arm(nil)

	m.arm([]AutoclickAction{{Trigger: "f8", Mode: "toggle", Button: "left", Interval: 0.01}})
	press := keyEvent{code: keyCodeMap["f8"], pressed: true}
	release := keyEvent{code: keyCodeMap["f8"], pressed: false}

	m.handleEvent(press) // on
	waitForClicks(t, rec, 1, time.Second)
	if rec.first() != mouseButtonMap["left"] {
		t.Errorf("clicked button = %d, want left", rec.first())
	}

	m.handleEvent(release) // release must not stop a toggle
	waitForClicks(t, rec, 3, time.Second)

	m.handleEvent(press) // off
	assertClicksStable(t, rec, 20*time.Millisecond, 50*time.Millisecond)
}

func TestAutoclickToggleIgnoresKeyRepeat(t *testing.T) {
	resetPauseState()
	rec := &clickRecorder{}
	m := newTestAutoclickManager(rec)
	defer m.arm(nil)

	m.arm([]AutoclickAction{{Trigger: "f8", Mode: "toggle", Button: "left", Interval: 0.01}})
	press := keyEvent{code: keyCodeMap["f8"], pressed: true}

	m.handleEvent(press) // on
	waitForClicks(t, rec, 1, time.Second)

	// Holding the key generates repeated press events; they must not toggle
	// the autoclicker back off.
	m.handleEvent(press)
	m.handleEvent(press)
	waitForClicks(t, rec, 3, time.Second)

	m.handleEvent(keyEvent{code: keyCodeMap["f8"], pressed: false})
	m.handleEvent(press) // off
	assertClicksStable(t, rec, 20*time.Millisecond, 50*time.Millisecond)
}

func TestAutoclickHold(t *testing.T) {
	resetPauseState()
	rec := &clickRecorder{}
	m := newTestAutoclickManager(rec)
	defer m.arm(nil)

	m.arm([]AutoclickAction{{Trigger: "mouse_side", Mode: "hold", Button: "right", Interval: 0.01}})
	press := keyEvent{code: mouseButtonMap["side"], pressed: true, button: true}
	release := keyEvent{code: mouseButtonMap["side"], pressed: false, button: true}

	m.handleEvent(press) // start clicking
	waitForClicks(t, rec, 1, time.Second)
	if rec.first() != mouseButtonMap["right"] {
		t.Errorf("clicked button = %d, want right", rec.first())
	}

	m.handleEvent(release) // stop clicking
	assertClicksStable(t, rec, 20*time.Millisecond, 50*time.Millisecond)
}

func TestAutoclickTriggerTypeMismatch(t *testing.T) {
	resetPauseState()
	rec := &clickRecorder{}
	m := newTestAutoclickManager(rec)
	defer m.arm(nil)

	// Trigger is the keyboard key "left" (arrow); a BTN_LEFT mouse press has
	// the same evdev code but must not trigger it.
	m.arm([]AutoclickAction{{Trigger: "left", Mode: "toggle", Interval: 0.01}})
	m.handleEvent(keyEvent{code: mouseButtonMap["left"], pressed: true, button: true})
	assertClicksStable(t, rec, 0, 30*time.Millisecond)
}

func TestAutoclickDisarmStops(t *testing.T) {
	resetPauseState()
	rec := &clickRecorder{}
	m := newTestAutoclickManager(rec)

	m.arm([]AutoclickAction{{Trigger: "f8", Mode: "toggle", Interval: 0.01}})
	m.handleEvent(keyEvent{code: keyCodeMap["f8"], pressed: true})
	waitForClicks(t, rec, 1, time.Second)

	m.arm(nil) // focus moved to an app without config
	assertClicksStable(t, rec, 20*time.Millisecond, 50*time.Millisecond)
}

func TestAutoclickPaused(t *testing.T) {
	resetPauseState()
	defer resetPauseState()
	rec := &clickRecorder{}
	m := newTestAutoclickManager(rec)
	defer m.arm(nil)

	pauseState.setActive("game", keyCodeMap["f9"], false)
	m.arm([]AutoclickAction{{Trigger: "f8", Mode: "toggle", Interval: 0.01}})

	pauseState.toggle(keyCodeMap["f9"]) // pause
	m.handleEvent(keyEvent{code: keyCodeMap["f8"], pressed: true})
	assertClicksStable(t, rec, 20*time.Millisecond, 50*time.Millisecond)

	pauseState.toggle(keyCodeMap["f9"]) // resume
	waitForClicks(t, rec, 1, time.Second)
}

func TestAutoclickAppOverridesGlobalByTrigger(t *testing.T) {
	resetPauseState()
	rec := &clickRecorder{}
	m := newTestAutoclickManager(rec)
	defer m.arm(nil)

	// Same trigger in global and app config: the app entry (armed last)
	// wins, here changing the click target from left to right.
	global := []AutoclickAction{{Trigger: "mouse_extra", Mode: "hold", Button: "left", Interval: 0.01}}
	app := []AutoclickAction{{Trigger: "mouse_extra", Mode: "hold", Button: "right", Interval: 0.01}}
	combined := append(append([]AutoclickAction{}, global...), app...)
	m.arm(combined)

	m.handleEvent(keyEvent{code: mouseButtonMap["extra"], pressed: true, button: true})
	waitForClicks(t, rec, 1, time.Second)
	if rec.first() != mouseButtonMap["right"] {
		t.Errorf("clicked button = %d, want right (app override)", rec.first())
	}
}

func TestAutoclickDefaults(t *testing.T) {
	resetPauseState()
	rec := &clickRecorder{}
	m := newTestAutoclickManager(rec)
	defer m.arm(nil)

	// No mode/button/interval: defaults to toggle, left button, 50ms.
	m.arm([]AutoclickAction{{Trigger: "f8"}})
	m.handleEvent(keyEvent{code: keyCodeMap["f8"], pressed: true})
	waitForClicks(t, rec, 1, time.Second)
	if rec.first() != mouseButtonMap["left"] {
		t.Errorf("default button = %d, want left", rec.first())
	}
}
