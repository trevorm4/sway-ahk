package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const (
	pidFile    = "/tmp/sway-ahk.pid"
	logFile    = "/tmp/sway-ahk.log"
	daemonFlag = "SWAY_AHK_DAEMON"
)

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

func notify(title, message string) {
	exec.Command("notify-send", title, message).Run()
}
