package main

import (
	"bufio"
	"encoding/json"
	"os/exec"
)

func monitorSwayFocus(focusChan chan<- string) {
	cmd := exec.Command("swaymsg", "-t", "subscribe", "-m", `["window"]`)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	if err := cmd.Start(); err != nil {
		return
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		var event struct {
			Change    string `json:"change"`
			Container struct {
				AppID            string `json:"app_id"`
				WindowProperties struct {
					Class string `json:"class"`
				} `json:"window_properties"`
			} `json:"container"`
		}

		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}

		if event.Change == "focus" {
			name := event.Container.AppID
			if name == "" {
				name = event.Container.WindowProperties.Class
			}
			if name != "" {
				focusChan <- name
			}
		}
	}
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

func queryFocusedApp() string {
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
