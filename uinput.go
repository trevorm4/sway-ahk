package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bendahl/uinput"
)

const (
	uinputPath    = "/dev/uinput"
	sysfsInputDir = "/sys/devices/virtual/input"
)

type uinputKeyboard struct {
	mu       sync.Mutex
	kb       uinput.Keyboard
	eventTag string
}

func openUinputKeyboard(name string) (*uinputKeyboard, error) {
	kb, err := uinput.CreateKeyboard(uinputPath, []byte(name))
	if err != nil {
		return nil, fmt.Errorf("create keyboard on %s: %w", uinputPath, err)
	}
	tag, tagErr := findEventTag(name)
	if tagErr != nil {
		log.Printf("uinput: %v (own-event filtering falls back to time-based guard)", tagErr)
	}
	return &uinputKeyboard{kb: kb, eventTag: tag}, nil
}

func (k *uinputKeyboard) EventTag() string {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.eventTag
}

func findEventTag(name string) (string, error) {
	entries, err := os.ReadDir(sysfsInputDir)
	if err != nil {
		return "", err
	}
	tag := ""
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "input") {
			continue
		}
		dir := filepath.Join(sysfsInputDir, entry.Name())
		data, err := os.ReadFile(filepath.Join(dir, "name"))
		if err != nil || strings.TrimSpace(string(data)) != name {
			continue
		}
		sub, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, node := range sub {
			if strings.HasPrefix(node.Name(), "event") {
				tag = node.Name()
			}
		}
	}
	if tag == "" {
		return "", fmt.Errorf("no sysfs event node found for %q", name)
	}
	return tag, nil
}

func (k *uinputKeyboard) TapKey(code int) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.kb.KeyPress(code)
}

func (k *uinputKeyboard) PressKey(code int) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.kb.KeyDown(code)
}

func (k *uinputKeyboard) ReleaseKey(code int) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.kb.KeyUp(code)
}

func (k *uinputKeyboard) Close() error {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.kb.Close()
}
