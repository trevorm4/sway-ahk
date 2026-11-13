package compositor

import (
	"bufio"
	"encoding/json"
	"log"
	"os/exec"
	"time"
)

// Sway implements Compositor for Sway (and compositors with a swaymsg-
// compatible IPC) using the swaymsg binary.
type Sway struct{}

// NewSway returns a Sway Compositor.
func NewSway() *Sway { return &Sway{} }

// Name returns "sway".
func (Sway) Name() string { return "sway" }

// MonitorFocus implements Compositor.
func (Sway) MonitorFocus(focusChan chan<- string) {
	for {
		if err := subscribeSwayFocus(focusChan); err != nil {
			log.Printf("sway focus monitor: %v, retrying in 5s", err)
		}
		time.Sleep(5 * time.Second)
	}
}

func subscribeSwayFocus(focusChan chan<- string) error {
	cmd := exec.Command("swaymsg", "-t", "subscribe", "-m", `["window"]`)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		if name, ok := parseSwayFocusEvent(scanner.Bytes()); ok {
			focusChan <- name
		}
	}
	return cmd.Wait()
}

// parseSwayFocusEvent extracts the focused window's app id (or X11 class)
// from a swaymsg window event. It reports ok=false for non-focus events and
// windows without a usable identifier.
func parseSwayFocusEvent(data []byte) (name string, ok bool) {
	var event struct {
		Change    string `json:"change"`
		Container struct {
			AppID            string `json:"app_id"`
			WindowProperties struct {
				Class string `json:"class"`
			} `json:"window_properties"`
		} `json:"container"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		return "", false
	}
	if event.Change != "focus" {
		return "", false
	}
	name = event.Container.AppID
	if name == "" {
		name = event.Container.WindowProperties.Class
	}
	return name, name != ""
}

// FocusedApp implements Compositor.
func (Sway) FocusedApp() string {
	out, err := exec.Command("swaymsg", "-t", "get_tree").Output()
	if err != nil {
		return ""
	}
	var root swayNode
	if err := json.Unmarshal(out, &root); err != nil {
		return ""
	}
	if name, ok := findFocusedNode(&root); ok {
		return name
	}
	return ""
}

type swayNode struct {
	Focused          bool   `json:"focused"`
	AppID            string `json:"app_id"`
	WindowProperties struct {
		Class string `json:"class"`
	} `json:"window_properties"`
	Nodes         []swayNode `json:"nodes"`
	FloatingNodes []swayNode `json:"floating_nodes"`
}

func findFocusedNode(node *swayNode) (string, bool) {
	if node.Focused {
		if node.AppID != "" {
			return node.AppID, true
		}
		if node.WindowProperties.Class != "" {
			return node.WindowProperties.Class, true
		}
		return "", false
	}
	for i := range node.Nodes {
		if name, ok := findFocusedNode(&node.Nodes[i]); ok {
			return name, true
		}
	}
	for i := range node.FloatingNodes {
		if name, ok := findFocusedNode(&node.FloatingNodes[i]); ok {
			return name, true
		}
	}
	return "", false
}
