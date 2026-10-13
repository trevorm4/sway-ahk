package main

import (
	"testing"
	"time"
)

func TestUinputTapKey(t *testing.T) {
	kb, err := openUinputKeyboard("sway-ahk-test")
	if err != nil {
		t.Skipf("cannot open %s, skipping integration test: %v", uinputPath, err)
	}
	defer kb.Close()

	if kb.EventTag() == "" {
		t.Fatalf("event tag was not resolved for the uinput device")
	}
	if err := kb.TapKey(keyCodeMap["e"]); err != nil {
		t.Fatalf("TapKey: %v", err)
	}
}

func TestUinputHoldKey(t *testing.T) {
	kb, err := openUinputKeyboard("sway-ahk-test")
	if err != nil {
		t.Skipf("cannot open %s, skipping integration test: %v", uinputPath, err)
	}
	defer kb.Close()

	if err := kb.PressKey(keyCodeMap["w"]); err != nil {
		t.Fatalf("PressKey: %v", err)
	}
	time.Sleep(500 * time.Millisecond)
	if err := kb.ReleaseKey(keyCodeMap["w"]); err != nil {
		t.Fatalf("ReleaseKey: %v", err)
	}
}
