## Intro

When using Sway, the thing I miss the most is basic scripting when gaming to automate pressing keys.

This repo provides a relatively simple script that allows configuring keys to press in intervals using [ydotool](https://github.com/ReimuNotMoe/ydotoo) and `swaymsg` for automation.


## Setup

Requires [ydotool](https://github.com/ReimuNotMoe/ydotoo) running in daemon mode

Build application via

`go build main.go -o build/sway-ahk main.go`

Run with

`./build/sway-ahk -config <path to config>`

Refer to the `config/` directory for an example config.

If you need help finding the class of your window, use the following command

```bash
sleep 2 && swaymsg -t get_tree | jq '.. | select(.type?) | select(.focused==true)'
```

and make sure to swap to the window that you are interested in within 2 seconds (or change the sleep duration if necessary)

If you wish to bind it in your sway config, then you would do the following

```
bindsym $mod+z exec sway-ahk -config ~/.config/sway/scripts/default.yaml
```
where `sway-ahk` must be in your `$PATH`
