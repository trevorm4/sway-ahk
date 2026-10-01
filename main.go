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
