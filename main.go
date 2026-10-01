package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	pidFile    = "/tmp/sway-ahk.pid"
	logFile    = "/tmp/sway-ahk.log"
	daemonFlag = "SWAY_AHK_DAEMON"
)

type KeyAction struct {
	Key      string  `yaml:"key"`
	Interval float64 `yaml:"interval"`
}

type RemapAction struct {
	Key   string   `yaml:"key"`
	To    []string `yaml:"to"`
	Delay int      `yaml:"delay"`
}

type AppConfig struct {
	AppClass string        `yaml:"app_class"`
	Keys     []KeyAction   `yaml:"keys"`
	Remaps   []RemapAction `yaml:"remaps"`
}

type Config struct {
	Apps []AppConfig `yaml:"apps"`
}

var keyCodeMap = map[string]int{
	"q": 16, "w": 17, "e": 18, "r": 19, "t": 20,
	"y": 21, "u": 22, "i": 23, "o": 24, "p": 25,
	"a": 30, "s": 31, "d": 32, "f": 33, "g": 34,
	"h": 35, "j": 36, "k": 37, "l": 38,
	"z": 44, "x": 45, "c": 46, "v": 47, "b": 48,
	"n": 49, "m": 50,
	"1": 2, "2": 3, "3": 4, "4": 5, "5": 6,
}

var configFilePath string

var virtualKeyboard *uinputKeyboard

var keyCodeNames = buildKeyCodeNames()

func buildKeyCodeNames() map[int]string {
	names := make(map[int]string, len(keyCodeMap))
	for name, code := range keyCodeMap {
		names[code] = name
	}
	return names
}

func keyName(code int) string {
	if name, ok := keyCodeNames[code]; ok {
		return name
	}
	return strconv.Itoa(code)
}

func keyNames(codes []int) string {
	parts := make([]string, 0, len(codes))
	for _, code := range codes {
		parts = append(parts, keyName(code))
	}
	return strings.Join(parts, " ")
}

const (
	defaultRemapDelayMs = 50
	injectedKeyGrace    = 150 * time.Millisecond
)

type remapManager struct {
	mu     sync.Mutex
	remaps map[int]RemapAction
	recent map[int]time.Time
}

var remapState = &remapManager{recent: make(map[int]time.Time)}

func (m *remapManager) setRemaps(actions []RemapAction) {
	built := make(map[int]RemapAction, len(actions))
	for _, action := range actions {
		code, ok := keyCodeMap[strings.ToLower(action.Key)]
		if !ok {
			log.Printf("remap: unknown key %q, skipping", action.Key)
			continue
		}
		if action.Delay <= 0 {
			action.Delay = defaultRemapDelayMs
		}
		built[code] = action
	}
	m.mu.Lock()
	m.remaps = built
	m.mu.Unlock()
}

func (m *remapManager) targetsFor(code int) ([]int, int, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.recent[code]; ok && time.Since(t) < injectedKeyGrace {
		return nil, 0, false
	}
	action, ok := m.remaps[code]
	if !ok {
		return nil, 0, false
	}
	targets := make([]int, 0, len(action.To))
	for _, key := range action.To {
		target, ok := keyCodeMap[strings.ToLower(key)]
		if !ok {
			log.Printf("remap: unknown target key %q, skipping", key)
			continue
		}
		targets = append(targets, target)
	}
	return targets, action.Delay, len(targets) > 0
}

func (m *remapManager) markInjected(code int) {
	m.mu.Lock()
	m.recent[code] = time.Now()
	m.mu.Unlock()
}

func main() {
	flag.StringVar(&configFilePath, "config", "sway-ahk-config.yaml", "Path to configuration file")
	flag.Parse()

	if os.Getenv(daemonFlag) == "1" {
		runDaemon()
		return
	}

	running, pid := getRunningPID()
	if running {
		fmt.Printf("Daemon is running (PID %d). Stopping...\n", pid)
		stopDaemon(pid)
		return
	}

	fmt.Println("Starting Sway AHK daemon...")
	startParent()
}

func getRunningPID() (bool, int) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false, 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false, 0
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false, 0
	}
	// Signal 0 checks if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil, pid
}

func stopDaemon(pid int) {
	process, _ := os.FindProcess(pid)
	err := process.Signal(syscall.SIGTERM)
	if err != nil {
		fmt.Printf("Error stopping process: %v\n", err)
	}
	os.Remove(pidFile)
	notify("Sway AHK", "Stopped")
}

func startParent() {
	absConfig, err := filepath.Abs(configFilePath)
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command(os.Args[0], "-config", absConfig)
	cmd.Env = append(os.Environ(), daemonFlag+"=1")

	// Create new session so it doesn't die with the non-daemon proc
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start daemon process: %v", err)
	}

	var pidOutput []byte
	err = os.WriteFile(pidFile, fmt.Appendf(pidOutput, "%d", cmd.Process.Pid), 0644)
	if err != nil {
		log.Fatalf("Failed to write PID file: %v", err)
	}

	notify("Sway AHK", "Started")
	os.Exit(0)
}

func runDaemon() {
	null, _ := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	os.Stdin = null
	os.Stdout = null
	os.Stderr = null

	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(f)
		defer f.Close()
	}
	log.Printf("Daemon initialized (PID: %d)", os.Getpid())

	config, err := loadConfig()
	if err != nil {
		log.Printf("Config error: %v", err)
		os.Remove(pidFile)
		return
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	vk, err := openUinputKeyboard("sway-ahk")
	if err != nil {
		log.Printf("failed to create uinput device (key injection disabled): %v", err)
	} else {
		virtualKeyboard = vk
		defer virtualKeyboard.Close()
		log.Println("uinput virtual keyboard created")
	}

	focusChan := make(chan string, 10)
	go monitorSwayFocus(focusChan)

	if appClass := queryFocusedApp(); appClass != "" {
		focusChan <- appClass
	}

	keyChan := make(chan int, 10)
	go monitorKeyEvents(keyChan)
	go handleKeyEvents(keyChan)

	var currentCancel context.CancelFunc
	var wg sync.WaitGroup

	for {
		select {
		case <-sigChan:
			if currentCancel != nil {
				currentCancel()
			}
			wg.Wait()
			os.Remove(pidFile)
			return

		case appClass := <-focusChan:
			if currentCancel != nil {
				currentCancel()
				wg.Wait()
			}

			appConfig := findAppConfig(config, appClass)
			if appConfig == nil {
				remapState.setRemaps(nil)
				continue
			}
			remapState.setRemaps(appConfig.Remaps)
			if len(appConfig.Remaps) > 0 {
				log.Printf("remaps active for %s (%d)", appClass, len(appConfig.Remaps))
			}

			ctx, cancel := context.WithCancel(context.Background())
			currentCancel = cancel

			for _, action := range appConfig.Keys {
				wg.Add(1)
				go pressKeyPeriodically(ctx, &wg, action)
			}
		}
	}
}

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

func monitorKeyEvents(keyChan chan<- int) {
	cmd := exec.Command("libinput", "debug-events", "--show-keycodes")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("key listener pipe error: %v", err)
		return
	}
	if err := cmd.Start(); err != nil {
		log.Printf("failed to start libinput debug-events (remaps require the libinput binary and permission to read input devices): %v", err)
		return
	}
	log.Println("key listener started")

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		if code, ok := parseKeyPressed(scanner.Text()); ok {
			keyChan <- code
		}
	}
}

func parseKeyPressed(line string) (int, bool) {
	if !strings.Contains(line, "KEYBOARD_KEY") {
		return 0, false
	}
	fields := strings.Fields(line)
	if len(fields) < 2 || fields[len(fields)-1] != "pressed" {
		return 0, false
	}
	for _, field := range fields {
		if len(field) > 2 && field[0] == '(' && field[len(field)-1] == ')' {
			if code, err := strconv.Atoi(field[1 : len(field)-1]); err == nil {
				return code, true
			}
		}
	}
	return 0, false
}

func handleKeyEvents(keyChan <-chan int) {
	for code := range keyChan {
		targets, delayMs, ok := remapState.targetsFor(code)
		if !ok {
			continue
		}
		log.Printf("remap: %s -> %s", keyName(code), keyNames(targets))
		for i, target := range targets {
			remapState.markInjected(target)
			pressKeyOnce(target)
			if i < len(targets)-1 {
				time.Sleep(time.Duration(delayMs) * time.Millisecond)
			}
		}
	}
}

func pressKeyPeriodically(ctx context.Context, wg *sync.WaitGroup, action KeyAction) {
	defer wg.Done()
	code, ok := keyCodeMap[strings.ToLower(action.Key)]
	if !ok {
		return
	}

	ticker := time.NewTicker(time.Duration(action.Interval * float64(time.Second)))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pressKeyOnce(code)
		}
	}
}

func pressKeyOnce(code int) {
	if virtualKeyboard == nil {
		return
	}
	if err := virtualKeyboard.TapKey(code); err != nil {
		log.Printf("uinput key tap failed: %v", err)
	}
}

func loadConfig() (*Config, error) {
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}
	var c Config
	err = yaml.Unmarshal(data, &c)
	return &c, err
}

func findAppConfig(config *Config, appClass string) *AppConfig {
	for _, app := range config.Apps {
		if strings.EqualFold(app.AppClass, appClass) {
			return &app
		}
	}
	return nil
}

func notify(title, message string) {
	exec.Command("notify-send", title, message).Run()
}
