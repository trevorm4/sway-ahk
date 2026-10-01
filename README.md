## Intro

When using Sway, the thing I miss the most is basic scripting when gaming to automate pressing keys.

This repo provides a relatively simple script that allows configuring keys to press in intervals using a virtual keyboard (uinput) and `swaymsg` for automation.


## Setup

Requires:

- `sway`
- the `libinput` binary in `$PATH` for key detection (bundled in the nix-built binary)
- write access to `/dev/uinput` for key injection — on NixOS this is granted to the active session user via ACL
- read access to `/dev/input` events for remaps — add your user to the `input` group, e.g. in NixOS:

```nix
users.users.<name>.extraGroups = [ "input" ];
```

then re-login for the group change to take effect.

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


## Remaps

A single key press can be mapped to multiple key presses for the focused app using `remaps`:

```yaml
apps:
  - app_class: "steam_app_1974050"
    remaps:
      - key: e          # pressing e...
        to: [q, w, e]   # ...also presses q, w, and e
        delay: 50       # optional: ms between each mapped key (default 50)
```

Notes:

- Key presses are detected with `libinput debug-events`, so the daemon needs permission to read input events (see Setup).
- Injected keys are sent via the daemon's own virtual keyboard; the daemon no longer needs ydotoold.
- The original key press still goes through to the application; the mapped keys are pressed in addition to it. Remaps only trigger on the initial press, not on key repeat.


## Development

Enter the dev shell with `nix develop`, then build with `go build` or via `nix build .#sway-ahk`.
