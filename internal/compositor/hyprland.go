package compositor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Hyprland implements Compositor for Hyprland. Focus events are read from
// the compositor's event socket (.socket2.sock) and the currently focused
// window is queried with hyprctl.
type Hyprland struct {
	instance string
}

// NewHyprland returns a Hyprland Compositor. It requires
// HYPRLAND_INSTANCE_SIGNATURE to be set, which it is for any process started
// from within a Hyprland session.
func NewHyprland() (*Hyprland, error) {
	instance := os.Getenv("HYPRLAND_INSTANCE_SIGNATURE")
	if instance == "" {
		return nil, fmt.Errorf("HYPRLAND_INSTANCE_SIGNATURE is not set (is Hyprland running?)")
	}
	return &Hyprland{instance: instance}, nil
}

// Name returns "hyprland".
func (Hyprland) Name() string { return "hyprland" }

// socket2Path locates the Hyprland event socket for this instance.
func (h Hyprland) socket2Path() string {
	candidates := make([]string, 0, 2)
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		candidates = append(candidates, filepath.Join(runtimeDir, "hypr", h.instance, ".socket2.sock"))
	}
	// Older Hyprland versions placed the sockets in /tmp.
	candidates = append(candidates, filepath.Join("/tmp", "hypr", h.instance, ".socket2.sock"))
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// MonitorFocus implements Compositor.
func (h Hyprland) MonitorFocus(focusChan chan<- string) {
	for {
		if err := h.subscribeFocus(focusChan); err != nil {
			log.Printf("hyprland focus monitor: %v, retrying in 5s", err)
		}
		time.Sleep(5 * time.Second)
	}
}

func (h Hyprland) subscribeFocus(focusChan chan<- string) error {
	path := h.socket2Path()
	if path == "" {
		return fmt.Errorf("event socket for instance %s not found", h.instance)
	}
	conn, err := net.Dial("unix", path)
	if err != nil {
		return err
	}
	defer conn.Close()
	log.Printf("hyprland focus monitor connected (%s)", path)

	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		if class, ok := parseHyprlandFocusEvent(line); ok {
			focusChan <- class
		}
	}
}

// parseHyprlandFocusEvent extracts the window class from an
// "activewindow>>CLASS,TITLE" socket event. It reports ok=false for other
// events and for events with an empty class (e.g. focusing an empty
// workspace).
func parseHyprlandFocusEvent(line string) (class string, ok bool) {
	payload, ok := strings.CutPrefix(strings.TrimSpace(line), "activewindow>>")
	if !ok {
		return "", false
	}
	class, _, _ = strings.Cut(payload, ",")
	return class, class != ""
}

// FocusedApp implements Compositor.
func (Hyprland) FocusedApp() string {
	out, err := exec.Command("hyprctl", "activewindow", "-j").Output()
	if err != nil {
		return ""
	}
	// With no window focused, hyprctl prints an empty object.
	var window struct {
		Class string `json:"class"`
	}
	if err := json.Unmarshal(out, &window); err != nil {
		return ""
	}
	return window.Class
}
