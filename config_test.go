package main

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseConfig(t *testing.T) {
	src := `
pause_key: "f9"
notifications:
  system: true
  sound: /path/to/sound.wav
apps:
  - app_class: "game"
    pause_key: "p"
    keys:
      - key: f
        interval: 5.0
      - key: w
        hold: true
        delay: 100
      - key: e
        hold: true
    remaps:
      - key: e
        to: [q, w, e]
        delay: 50
`
	var c Config
	if err := yaml.Unmarshal([]byte(src), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.PauseKey != "f9" {
		t.Errorf("pause_key = %q, want %q", c.PauseKey, "f9")
	}
	if !c.Notifications.System || c.Notifications.Sound != "/path/to/sound.wav" {
		t.Errorf("notifications = %+v", c.Notifications)
	}
	if len(c.Apps) != 1 || c.Apps[0].AppClass != "game" || c.Apps[0].PauseKey != "p" {
		t.Errorf("apps = %+v", c.Apps)
	}
	if len(c.Apps[0].Keys) != 3 {
		t.Fatalf("keys = %+v", c.Apps[0].Keys)
	}
	if c.Apps[0].Keys[0].Key != "f" || c.Apps[0].Keys[0].Interval != 5.0 || c.Apps[0].Keys[0].Hold {
		t.Errorf("keys[0] = %+v", c.Apps[0].Keys[0])
	}
	if c.Apps[0].Keys[1].Key != "w" || !c.Apps[0].Keys[1].Hold || c.Apps[0].Keys[1].Delay != 100 {
		t.Errorf("keys[1] = %+v", c.Apps[0].Keys[1])
	}
	if c.Apps[0].Keys[2].Key != "e" || !c.Apps[0].Keys[2].Hold || c.Apps[0].Keys[2].Delay != 0 {
		t.Errorf("keys[2] = %+v", c.Apps[0].Keys[2])
	}
	if len(c.Apps[0].Remaps) != 1 || c.Apps[0].Remaps[0].Key != "e" || len(c.Apps[0].Remaps[0].To) != 3 {
		t.Errorf("remaps = %+v", c.Apps[0].Remaps)
	}
}
