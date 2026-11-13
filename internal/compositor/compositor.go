// Package compositor abstracts window-focus interactions with Wayland
// compositors so the daemon can run on Sway, Hyprland, or others.
package compositor

import (
	"fmt"
	"os"
	"strings"
)

// Compositor reports which application window currently has keyboard focus.
type Compositor interface {
	// Name returns the compositor identifier, e.g. "sway" or "hyprland".
	Name() string
	// MonitorFocus streams the app class of each newly focused window to
	// focusChan. It blocks forever, retrying internally on transient errors,
	// and is meant to run in its own goroutine.
	MonitorFocus(focusChan chan<- string)
	// FocusedApp returns the app class of the currently focused window, or
	// "" if it cannot be determined.
	FocusedApp() string
}

// Detect returns a Compositor based on preferred: "sway", "hyprland", or
// "auto"/"" to detect the running compositor from the environment.
func Detect(preferred string) (Compositor, error) {
	switch strings.ToLower(preferred) {
	case "sway":
		return NewSway(), nil
	case "hyprland":
		return NewHyprland()
	case "", "auto":
		return detectFromEnv()
	default:
		return nil, fmt.Errorf("unknown compositor %q (want \"sway\", \"hyprland\", or \"auto\")", preferred)
	}
}

func detectFromEnv() (Compositor, error) {
	if os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") != "" {
		return NewHyprland()
	}
	if os.Getenv("SWAYSOCK") != "" {
		return NewSway(), nil
	}
	desktop := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))
	switch {
	case strings.Contains(desktop, "hyprland"):
		return NewHyprland()
	case strings.Contains(desktop, "sway"):
		return NewSway(), nil
	}
	return nil, fmt.Errorf("no supported compositor detected " +
		"(HYPRLAND_INSTANCE_SIGNATURE, SWAYSOCK, and XDG_CURRENT_DESKTOP are all unset)")
}
