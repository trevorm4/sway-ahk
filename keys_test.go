package main

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

type recordingInjector struct {
	mu     sync.Mutex
	events []string
}

func (r *recordingInjector) record(event string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}

func (r *recordingInjector) snapshot() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return strings.Join(r.events, " ")
}

func (r *recordingInjector) TapKey(code int) error {
	r.record("tap " + keyName(code))
	return nil
}

func (r *recordingInjector) PressKey(code int) error {
	r.record("down " + keyName(code))
	return nil
}

func (r *recordingInjector) ReleaseKey(code int) error {
	r.record("up " + keyName(code))
	return nil
}

func resetPauseState() {
	pauseState.mu.Lock()
	pauseState.appClass = ""
	pauseState.pauseKey = 0
	pauseState.appScoped = false
	pauseState.global = false
	pauseState.apps = make(map[string]bool)
	pauseState.mu.Unlock()
	holdState.disarm()
}

func TestMonitorHoldsOrdering(t *testing.T) {
	savedInjector := virtualKeyboard
	virtualKeyboard = &recordingInjector{}
	defer func() { virtualKeyboard = savedInjector }()

	resetPauseState()
	defer resetPauseState()

	pauseState.setActive("test", keyCodeMap["f9"], true)

	actions := []KeyAction{
		{Key: "w", Hold: true, Delay: 50},
		{Key: "e", Hold: true, Delay: 50},
		{Key: "r", Hold: true},
		{Key: "f", Interval: 5.0},
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go monitorHolds(ctx, &wg, actions)

	rec := virtualKeyboard.(*recordingInjector)
	startOrder := "down w down e down r"
	waitForEvents(t, rec, startOrder, 2*time.Second)

	if !pauseState.toggle(keyCodeMap["f9"]) {
		t.Fatal("pause key should be handled")
	}
	pauseOrder := startOrder + " up w up e up r"
	waitForEvents(t, rec, pauseOrder, 2*time.Second)

	if !pauseState.toggle(keyCodeMap["f9"]) {
		t.Fatal("pause key should be handled")
	}
	resumeOrder := pauseOrder + " down w down e down r"
	waitForEvents(t, rec, resumeOrder, 2*time.Second)

	cancel()
	wg.Wait()

	want := resumeOrder + " up w up e up r"
	if got := rec.snapshot(); got != want {
		t.Errorf("unexpected event sequence:\n got: %s\nwant: %s", got, want)
	}
}

func waitForEvents(t *testing.T, rec *recordingInjector, want string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if rec.snapshot() == want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q, got %q", want, rec.snapshot())
}

func waitForSubstring(t *testing.T, rec *recordingInjector, want string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(rec.snapshot(), want) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q, got %q", want, rec.snapshot())
}

func waitForCount(t *testing.T, rec *recordingInjector, event string, want int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Count(rec.snapshot(), event) >= want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d occurrences of %q, got %q", want, event, rec.snapshot())
}

func clearInjectedMarks() {
	remapState.mu.Lock()
	remapState.recent = make(map[int]time.Time)
	remapState.mu.Unlock()
}

func TestHoldReassert(t *testing.T) {
	savedInjector := virtualKeyboard
	rec := &recordingInjector{}
	virtualKeyboard = rec
	defer func() { virtualKeyboard = savedInjector }()

	resetPauseState()
	defer resetPauseState()

	actions := []KeyAction{{Key: "w", Hold: true, Reassert: 200}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(1)
	go monitorHolds(ctx, &wg, actions)

	waitForCount(t, rec, "down w", 1, 2*time.Second)

	holdState.notifyReassert(keyCodeMap["w"])
	waitForCount(t, rec, "down w", 2, 2*time.Second)

	keyChan := make(chan keyEvent, 10)
	handleDone := make(chan struct{})
	go func() {
		handleKeyEvents(keyChan)
		close(handleDone)
	}()

	clearInjectedMarks()
	keyChan <- keyEvent{code: keyCodeMap["w"], pressed: false}
	waitForCount(t, rec, "down w", 3, 2*time.Second)

	clearInjectedMarks()
	pressKeyOnce(keyCodeMap["w"])
	waitForSubstring(t, rec, "tap w", 2*time.Second)
	waitForCount(t, rec, "down w", 4, 2*time.Second)

	waitForCount(t, rec, "down w", 5, 3*time.Second)

	close(keyChan)
	<-handleDone
	cancel()
	wg.Wait()

	if !strings.Contains(rec.snapshot(), "up w") {
		t.Errorf("hold was not released on stop: %s", rec.snapshot())
	}
}
