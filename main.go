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

// --- Configuration Types ---

type KeyAction struct {
	Key      string  `yaml:"key"`
	Interval float64 `yaml:"interval"`
}

type AppConfig struct {
	AppClass string      `yaml:"app_class"`
	Keys     []KeyAction `yaml:"keys"`
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

// --- Entry Point ---

func main() {
	flag.StringVar(&configFilePath, "config", "sway-ahk-config.yaml", "Path to configuration file")
	flag.Parse()

	// 1. Check if we are already the daemonized child
	if os.Getenv(daemonFlag) == "1" {
		runDaemon()
		return
	}

	// 2. Check if a daemon is already running globally
	running, pid := getRunningPID()
	if running {
		fmt.Printf("Daemon is running (PID %d). Stopping...\n", pid)
		stopDaemon(pid)
		return
	}

	// 3. Start the daemon (Fork/Exec)
	fmt.Println("Starting Sway AHK daemon...")
	startParent()
}

// --- Process Management ---

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

	// Create new session so it doesn't die with the terminal
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start daemon process: %v", err)
	}

	// Parent writes the PID file for the child
	var pidOutput []byte
	err = os.WriteFile(pidFile, fmt.Appendf(pidOutput, "%d", cmd.Process.Pid), 0644)
	if err != nil {
		log.Fatalf("Failed to write PID file: %v", err)
	}

	notify("Sway AHK", "Started")
	os.Exit(0)
}

// --- Daemon Logic ---

func runDaemon() {
	// Redirect I/O to /dev/null to fully detach
	null, _ := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	os.Stdin = null
	os.Stdout = null
	os.Stderr = null

	// Setup Logging
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

	focusChan := make(chan string, 10)
	go monitorSwayFocus(focusChan)

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
				continue
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

// --- Helpers (Sway, Config, Keyboard) ---

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
			// ydotool press (1) and release (0)
			exec.Command("ydotool", "key", fmt.Sprintf("%d:1", code), fmt.Sprintf("%d:0", code)).Run()
		}
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
