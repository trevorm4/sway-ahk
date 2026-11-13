package main

import (
	"errors"
	"log"
	"os/exec"
)

var notifyConfig NotifyConfig

// displayName is the human-facing app name used in notifications.
var displayName = "Sway AHK"

func displayNameFor(compositor string) string {
	switch compositor {
	case "hyprland":
		return "Hyprland AHK"
	default:
		return "Sway AHK"
	}
}

func notify(title, message string) {
	if err := exec.Command("notify-send", title, message).Run(); err != nil {
		log.Printf("notify-send failed: %v", err)
	}
}

func notifyPauseState(paused bool, scope string) {
	if notifyConfig.System {
		if paused {
			go notify(displayName, "paused "+scope)
		} else {
			go notify(displayName, "resumed "+scope)
		}
	}
	if notifyConfig.Sound != "" {
		go playSound(notifyConfig.Sound)
	}
}

func playSound(path string) {
	players := [][]string{
		{"pw-play", path},
		{"paplay", path},
		{"ffplay", "-nodisp", "-autoexit", "-loglevel", "quiet", path},
		{"mpv", "--really-quiet", path},
	}
	var err error
	for _, player := range players {
		cmd := exec.Command(player[0], player[1:]...)
		err = cmd.Run()
		if err == nil {
			return
		}
		if !errors.Is(err, exec.ErrNotFound) {
			log.Printf("sound player %s failed: %v", player[0], err)
			return
		}
	}
	log.Printf("no sound player found (tried pw-play, paplay, ffplay, mpv), cannot play %s", path)
}
