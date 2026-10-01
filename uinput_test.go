package main

import (
	"testing"
	"unsafe"
)

func TestUinputStructSizes(t *testing.T) {
	if unsafe.Sizeof(inputEvent{}) != 24 {
		t.Errorf("inputEvent size = %d, want 24", unsafe.Sizeof(inputEvent{}))
	}
	if unsafe.Sizeof(uinputID{}) != 8 {
		t.Errorf("uinputID size = %d, want 8", unsafe.Sizeof(uinputID{}))
	}
	if unsafe.Sizeof(uinputUserDev{}) != 1116 {
		t.Errorf("uinputUserDev size = %d, want 1116", unsafe.Sizeof(uinputUserDev{}))
	}
}

func TestUinputTapKey(t *testing.T) {
	kb, err := openUinputKeyboard("sway-ahk-test")
	if err != nil {
		t.Skipf("cannot open %s, skipping integration test: %v", uinputPath, err)
	}
	defer kb.Close()

	if err := kb.TapKey(keyCodeMap["e"]); err != nil {
		t.Fatalf("TapKey: %v", err)
	}
}
