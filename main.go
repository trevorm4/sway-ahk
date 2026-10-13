package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

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
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
		log.Printf("failed to write PID file: %v", err)
	}

	config, err := loadConfig()
	if err != nil {
		log.Printf("Config error: %v", err)
		os.Remove(pidFile)
		return
	}
	globalPauseKey := resolveKeyCode(config.PauseKey)
	notifyConfig = config.Notifications

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	vk, err := openUinputKeyboard("sway-ahk")
	if err != nil {
		log.Printf("failed to create uinput device (key injection disabled): %v", err)
	} else {
		virtualKeyboard = vk
		ownEventTag = vk.EventTag()
		defer vk.Close()
		if ownEventTag != "" {
			log.Printf("uinput virtual keyboard created (%s)", ownEventTag)
		} else {
			log.Println("uinput virtual keyboard created")
		}
	}

	focusChan := make(chan string, 10)
	go monitorSwayFocus(focusChan)

	if appClass := queryFocusedApp(); appClass != "" {
		focusChan <- appClass
	}

	keyChan := make(chan keyEvent, 10)
	go monitorKeyEvents(keyChan)
	go handleKeyEvents(keyChan)

	var currentCancel context.CancelFunc
	var wg sync.WaitGroup
	activeClass := ""

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
			if appClass == activeClass {
				log.Printf("focus: %s (unchanged)", appClass)
				continue
			}
			log.Printf("focus: %s", appClass)
			activeClass = appClass

			if currentCancel != nil {
				currentCancel()
				wg.Wait()
			}

			appConfig := findAppConfig(config, appClass)
			if appConfig == nil {
				remapState.setRemaps(nil)
				pauseState.setActive(appClass, globalPauseKey, false)
				continue
			}
			remapState.setRemaps(appConfig.Remaps)
			if len(appConfig.Remaps) > 0 {
				log.Printf("remaps active for %s (%d)", appClass, len(appConfig.Remaps))
			}

			pauseKey, appScoped := globalPauseKey, false
			if key := resolveKeyCode(appConfig.PauseKey); key != 0 {
				pauseKey, appScoped = key, true
			}
			pauseState.setActive(appClass, pauseKey, appScoped)

			ctx, cancel := context.WithCancel(context.Background())
			currentCancel = cancel

			holdCount := 0
			for _, action := range appConfig.Keys {
				if action.Hold {
					holdCount++
				}
			}
			if holdCount > 0 {
				log.Printf("holds active for %s (%d)", appClass, holdCount)
				wg.Add(1)
				go monitorHolds(ctx, &wg, appConfig.Keys)
			}
			for _, action := range appConfig.Keys {
				if action.Hold {
					continue
				}
				wg.Add(1)
				go pressKeyPeriodically(ctx, &wg, action)
			}
		}
	}
}
