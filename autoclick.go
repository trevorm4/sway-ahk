package main

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"
)

const defaultClickInterval = 50 * time.Millisecond

// mouseButtonMap maps mouse button names to evdev codes (BTN_*). Buttons are
// usable as autoclick triggers (with the "mouse_" prefix) and, for
// left/right/middle, as click targets.
var mouseButtonMap = map[string]int{
	"left":    0x110, // BTN_LEFT
	"right":   0x111, // BTN_RIGHT
	"middle":  0x112, // BTN_MIDDLE
	"side":    0x113, // BTN_SIDE
	"extra":   0x114, // BTN_EXTRA
	"forward": 0x115, // BTN_FORWARD
	"back":    0x116, // BTN_BACK
}

// mouseInjector is the subset of the uinput mouse used for clicking.
type mouseInjector interface {
	ClickButton(code int) error
}

var virtualMouse mouseInjector

// ownMouseEventTag is the sysfs event node of our uinput mouse, used to
// filter out our own injected clicks.
var ownMouseEventTag string

// clickTrigger identifies an input that starts/stops an autoclicker: either
// a keyboard key or a mouse button (evdev codes overlap between the two
// namespaces, so the event type is part of the identity).
type clickTrigger struct {
	code   int
	button bool
}

type resolvedClick struct {
	button   int
	interval time.Duration
	toggle   bool
}

type autoclickManager struct {
	mu      sync.Mutex
	actions map[clickTrigger]resolvedClick
	held    map[clickTrigger]bool
	running map[clickTrigger]bool
	cancel  map[clickTrigger]context.CancelFunc
	click   func(code int) // injectable for tests
}

var autoclickState = newAutoclickManager()

func newAutoclickManager() *autoclickManager {
	return &autoclickManager{click: clickMouseOnce}
}

// resolveClickTrigger parses a trigger name: "mouse_<button>" for mouse
// buttons (the prefix is required since e.g. "left" is also the arrow key),
// otherwise a keyboard key name.
func resolveClickTrigger(name string) (clickTrigger, bool) {
	lower := strings.ToLower(name)
	if btn, ok := strings.CutPrefix(lower, "mouse_"); ok {
		code, ok := mouseButtonMap[btn]
		return clickTrigger{code: code, button: true}, ok
	}
	code, ok := keyCodeMap[lower]
	return clickTrigger{code: code}, ok
}

func (t clickTrigger) String() string {
	if t.button {
		for name, code := range mouseButtonMap {
			if code == t.code {
				return "mouse_" + name
			}
		}
	}
	return keyName(t.code)
}

func isClickableButton(code int) bool {
	return code == mouseButtonMap["left"] || code == mouseButtonMap["right"] || code == mouseButtonMap["middle"]
}

// arm installs the autoclickers for the focused app, stopping any previously
// running click loops. Pass nil to disarm (app without config).
func (m *autoclickManager) arm(actions []AutoclickAction) {
	m.mu.Lock()
	for _, cancel := range m.cancel {
		cancel()
	}
	m.actions = make(map[clickTrigger]resolvedClick, len(actions))
	m.held = make(map[clickTrigger]bool)
	m.running = make(map[clickTrigger]bool)
	m.cancel = make(map[clickTrigger]context.CancelFunc)
	m.mu.Unlock()

	for _, action := range actions {
		trigger, ok := resolveClickTrigger(action.Trigger)
		if !ok {
			log.Printf("autoclick: unknown trigger %q, skipping", action.Trigger)
			continue
		}
		button := mouseButtonMap["left"]
		if action.Button != "" {
			b, ok := mouseButtonMap[strings.ToLower(action.Button)]
			if !ok {
				log.Printf("autoclick: unknown button %q, skipping", action.Button)
				continue
			}
			button = b
		}
		if !isClickableButton(button) {
			log.Printf("autoclick: button %q cannot be clicked (only left/right/middle), skipping", action.Button)
			continue
		}
		interval := defaultClickInterval
		if action.Interval > 0 {
			interval = time.Duration(action.Interval * float64(time.Second))
		}
		toggle := true
		switch strings.ToLower(action.Mode) {
		case "", "toggle":
		case "hold":
			toggle = false
		default:
			log.Printf("autoclick: unknown mode %q, using toggle", action.Mode)
		}
		m.mu.Lock()
		m.actions[trigger] = resolvedClick{button: button, interval: interval, toggle: toggle}
		m.mu.Unlock()
	}
}

// handleEvent processes a key/button event; call for both presses and
// releases. Trigger events still pass through to the application (we cannot
// swallow input), same as the pause key.
func (m *autoclickManager) handleEvent(ev keyEvent) {
	trigger := clickTrigger{code: ev.code, button: ev.button}
	m.mu.Lock()
	defer m.mu.Unlock()
	action, ok := m.actions[trigger]
	if !ok {
		return
	}
	if ev.pressed {
		if m.held[trigger] {
			return // keyboard auto-repeat
		}
		m.held[trigger] = true
		if action.toggle && m.running[trigger] {
			m.stopLocked(trigger)
			log.Printf("autoclick: %s off", trigger)
			return
		}
		m.startLocked(trigger, action)
		log.Printf("autoclick: %s on (button %s every %v)", trigger, keyName(action.button), action.interval)
		return
	}
	m.held[trigger] = false
	if !action.toggle {
		m.stopLocked(trigger)
		log.Printf("autoclick: %s off (released)", trigger)
	}
}

func (m *autoclickManager) startLocked(trigger clickTrigger, action resolvedClick) {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel[trigger] = cancel
	m.running[trigger] = true
	click := m.click
	go func() {
		if !pauseState.paused() {
			click(action.button)
		}
		ticker := time.NewTicker(action.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if pauseState.paused() {
					continue
				}
				click(action.button)
			}
		}
	}()
}

func (m *autoclickManager) stopLocked(trigger clickTrigger) {
	if cancel, ok := m.cancel[trigger]; ok {
		cancel()
		delete(m.cancel, trigger)
	}
	delete(m.running, trigger)
}

// clickMouseOnce taps a mouse button via the uinput virtual mouse.
func clickMouseOnce(code int) {
	if virtualMouse == nil {
		return
	}
	remapState.markInjected(code)
	if err := virtualMouse.ClickButton(code); err != nil {
		log.Printf("uinput mouse click failed: %v", err)
	}
}
