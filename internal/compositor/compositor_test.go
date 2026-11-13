package compositor

import (
	"testing"
)

func TestDetectExplicit(t *testing.T) {
	comp, err := Detect("sway")
	if err != nil || comp.Name() != "sway" {
		t.Errorf("Detect(sway) = (%v, %v), want sway", comp, err)
	}

	t.Setenv("HYPRLAND_INSTANCE_SIGNATURE", "deadbeef")
	comp, err = Detect("hyprland")
	if err != nil || comp.Name() != "hyprland" {
		t.Errorf("Detect(hyprland) = (%v, %v), want hyprland", comp, err)
	}
}

func TestDetectExplicitHyprlandWithoutInstance(t *testing.T) {
	t.Setenv("HYPRLAND_INSTANCE_SIGNATURE", "")
	if _, err := Detect("hyprland"); err == nil {
		t.Error("Detect(hyprland) without HYPRLAND_INSTANCE_SIGNATURE should fail")
	}
}

func TestDetectAuto(t *testing.T) {
	t.Run("hyprland", func(t *testing.T) {
		t.Setenv("HYPRLAND_INSTANCE_SIGNATURE", "deadbeef")
		t.Setenv("SWAYSOCK", "")
		t.Setenv("XDG_CURRENT_DESKTOP", "")
		comp, err := Detect("auto")
		if err != nil || comp.Name() != "hyprland" {
			t.Errorf("Detect(auto) = (%v, %v), want hyprland", comp, err)
		}
	})

	t.Run("sway", func(t *testing.T) {
		t.Setenv("HYPRLAND_INSTANCE_SIGNATURE", "")
		t.Setenv("SWAYSOCK", "/run/user/1000/sway-ipc.sock")
		t.Setenv("XDG_CURRENT_DESKTOP", "")
		comp, err := Detect("auto")
		if err != nil || comp.Name() != "sway" {
			t.Errorf("Detect(auto) = (%v, %v), want sway", comp, err)
		}
	})

	t.Run("desktop fallback", func(t *testing.T) {
		t.Setenv("HYPRLAND_INSTANCE_SIGNATURE", "")
		t.Setenv("SWAYSOCK", "")
		t.Setenv("XDG_CURRENT_DESKTOP", "sway")
		comp, err := Detect("auto")
		if err != nil || comp.Name() != "sway" {
			t.Errorf("Detect(auto) = (%v, %v), want sway", comp, err)
		}
	})

	t.Run("nothing set", func(t *testing.T) {
		t.Setenv("HYPRLAND_INSTANCE_SIGNATURE", "")
		t.Setenv("SWAYSOCK", "")
		t.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
		if _, err := Detect("auto"); err == nil {
			t.Error("Detect(auto) with no compositor env should fail")
		}
	})
}

func TestDetectUnknown(t *testing.T) {
	if _, err := Detect("i3"); err == nil {
		t.Error("Detect(i3) should fail")
	}
}
