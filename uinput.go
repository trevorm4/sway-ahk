package main

import (
	"fmt"
	"sync"

	"github.com/bendahl/uinput"
)

const uinputPath = "/dev/uinput"

type uinputKeyboard struct {
	mu sync.Mutex
	kb uinput.Keyboard
}

func openUinputKeyboard(name string) (*uinputKeyboard, error) {
	kb, err := uinput.CreateKeyboard(uinputPath, []byte(name))
	if err != nil {
		return nil, fmt.Errorf("create keyboard on %s: %w", uinputPath, err)
	}
	return &uinputKeyboard{kb: kb}, nil
}

func (k *uinputKeyboard) TapKey(code int) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.kb.KeyPress(code)
}

func (k *uinputKeyboard) Close() error {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.kb.Close()
}
