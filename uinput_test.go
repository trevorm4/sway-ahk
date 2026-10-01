package main

import "testing"

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
