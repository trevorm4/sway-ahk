package main

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultRemapDelayMs = 50
	injectedKeyGrace    = 150 * time.Millisecond
)

type remapManager struct {
	mu     sync.Mutex
	remaps map[int]RemapAction
	recent map[int]time.Time
}

var remapState = &remapManager{recent: make(map[int]time.Time)}

func (m *remapManager) setRemaps(actions []RemapAction) {
	built := make(map[int]RemapAction, len(actions))
	for _, action := range actions {
		code, ok := keyCodeMap[strings.ToLower(action.Key)]
		if !ok {
			log.Printf("remap: unknown key %q, skipping", action.Key)
			continue
		}
		if action.Delay <= 0 {
			action.Delay = defaultRemapDelayMs
		}
		built[code] = action
	}
	m.mu.Lock()
	m.remaps = built
	m.mu.Unlock()
}

func (m *remapManager) targetsFor(code int) ([]int, int, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.recent[code]; ok && time.Since(t) < injectedKeyGrace {
		return nil, 0, false
	}
	action, ok := m.remaps[code]
	if !ok {
		return nil, 0, false
	}
	targets := make([]int, 0, len(action.To))
	for _, key := range action.To {
		target, ok := keyCodeMap[strings.ToLower(key)]
		if !ok {
			log.Printf("remap: unknown target key %q, skipping", key)
			continue
		}
		targets = append(targets, target)
	}
	return targets, action.Delay, len(targets) > 0
}

func (m *remapManager) markInjected(code int) {
	m.mu.Lock()
	m.recent[code] = time.Now()
	m.mu.Unlock()
}

func (m *remapManager) recentlyInjected(code int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.recent[code]
	return ok && time.Since(t) < injectedKeyGrace
}

type keyEvent struct {
	code    int
	pressed bool
	device  string
}

func monitorKeyEvents(keyChan chan<- keyEvent) {
	for {
		if err := subscribeKeyEvents(keyChan); err != nil {
			log.Printf("key listener: %v, retrying in 5s", err)
		}
		time.Sleep(5 * time.Second)
	}
}

func subscribeKeyEvents(keyChan chan<- keyEvent) error {
	cmd := exec.Command("libinput", "debug-events", "--show-keycodes")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("libinput debug-events (remaps require the libinput binary and permission to read input devices): %w", err)
	}
	log.Println("key listener started")

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		if ev, ok := parseKeyEvent(scanner.Text()); ok {
			keyChan <- ev
		}
	}
	return cmd.Wait()
}

func parseKeyEvent(line string) (keyEvent, bool) {
	if !strings.Contains(line, "KEYBOARD_KEY") {
		return keyEvent{}, false
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return keyEvent{}, false
	}
	ev := keyEvent{device: strings.TrimPrefix(fields[0], "-")}
	switch fields[len(fields)-1] {
	case "pressed":
		ev.pressed = true
	case "released":
		ev.pressed = false
	default:
		return keyEvent{}, false
	}
	for _, field := range fields {
		if len(field) > 2 && field[0] == '(' && field[len(field)-1] == ')' {
			if code, err := strconv.Atoi(field[1 : len(field)-1]); err == nil {
				ev.code = code
				return ev, true
			}
		}
	}
	return keyEvent{}, false
}

func isOwnEvent(ev keyEvent) bool {
	if ownEventTag != "" {
		return ev.device == ownEventTag
	}
	return remapState.recentlyInjected(ev.code)
}

func handleKeyEvents(keyChan <-chan keyEvent) {
	for ev := range keyChan {
		if isOwnEvent(ev) {
			continue
		}
		if !ev.pressed {
			if holdState.isHeld(ev.code) {
				holdState.notifyReassert(ev.code)
			}
			continue
		}
		if pauseState.toggle(ev.code) {
			continue
		}
		if pauseState.paused() {
			continue
		}
		targets, delayMs, ok := remapState.targetsFor(ev.code)
		if !ok {
			continue
		}
		log.Printf("remap: %s -> %s", keyName(ev.code), keyNames(targets))
		for i, target := range targets {
			pressKeyOnce(target)
			if i < len(targets)-1 {
				time.Sleep(time.Duration(delayMs) * time.Millisecond)
			}
		}
	}
}
