## Intro

When using Sway, the thing I miss the most is basic scripting when gaming to automate pressing keys.

This repo provides a relatively simple script that allows configuring keys to press in intervals using [ydotool](https://github.com/ReimuNotMoe/ydotoo) and `swaymsg` for automation.


## Setup

Requires [ydotool](https://github.com/ReimuNotMoe/ydotoo) running in daemon mode

Build application via

`go build main.go -o sway-ahk main.go`

Run with

`./sway-ahk -config <path to config>`

Refer to the `config/` directory for an example config.

If you need help finding the class of your window, use the following command

```bash
sleep 2 && swaymsg -t get_tree | jq '.. | select(.type?) | select(.focused==true)'
```

and make sure to swap to the window that you are interested in within 2 seconds (or change the sleep duration if necessary)

