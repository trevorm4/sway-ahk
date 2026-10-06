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

`go build -o build/sway-ahk .`

Run with

`./build/sway-ahk -config <path to config>`

Refer to the `config/` directory for an example config.

Key names are the Linux key codes (`KEY_*` from `input-event-codes.h`) without the `KEY_` prefix, lowercase — e.g. `q`, `f1`, `leftshift`, `kpenter`, `space`.

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


## Hold keys

Instead of tapping a key at an interval, a key can be held down continuously while the app is focused:

```yaml
apps:
  - app_class: "my_game"
    keys:
      - key: w
        hold: true     # w is held down while this app is focused
        delay: 100     # optional: ms to wait after this key goes down before the next hold starts
      - key: e
        hold: true
      - key: f
        interval: 5.0  # f is tapped every 5 seconds
```

Hold keys go down in config order, and keep that order when pausing and resuming. Held keys are released when the app loses focus, when the daemon stops, and while paused — pressing the pause key toggles them down/up along with everything else. Use `delay` to space out the key-downs for games that sample key state once per frame instead of processing key events.

For XWayland games, a key-up from *any* device releases that key globally — so physically pressing a held key (or having it as a remap target) breaks the hold even though the virtual keyboard still holds it down. The daemon watches for that and automatically re-presses held keys released elsewhere (logged as `hold: <key> re-asserted (released elsewhere)`). If a game still loses held keys (some clear key state internally on focus hiccups), set `reassert` to re-press the key every N milliseconds:

```yaml
    keys:
      - key: w
        hold: true
        reassert: 250   # optional: re-press w every 250ms; only use for keys where repeated presses are harmless (e.g. movement)
```


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


## Pausing

Interval presses and remaps can be toggled with a pause key, either globally or per app (per-app overrides global):

```yaml
pause_key: "f9" # optional: toggles everything, works regardless of focused window

apps:
  - app_class: "steam_app_1974050"
    pause_key: "p" # optional: toggles this app only, while it is focused
```

Notes:

- While paused, remaps don't trigger and interval keys don't press; timers keep running, so unpausing takes effect immediately.
- Pause state persists while the daemon runs, including across focus changes.
- The pause key press still goes through to the application, so pick a key that is harmless in game. Avoid using the pause key as a remap target.

Pause toggles can be announced via a system notification and/or a sound:

```yaml
notifications:
  system: true                    # optional: notify-send on pause/resume
  sound: "/path/to/sound.wav"      # optional: played via pw-play, paplay, ffplay, or mpv
```


## Development

Enter the dev shell with `nix develop`, then build with `go build` or via `nix build .#sway-ahk`.


## Systemd service

The flake exposes a NixOS module (`nixosModules.default`) that runs the daemon as a systemd user service, started with `graphical-session.target`:

```nix
# flake input
sway-ahk = {
  url = "git+file:/home/trevor/Repos/sway-ahk";
  inputs.nixpkgs.follows = "nixpkgs";
};

# in your NixOS config
nixpkgs.overlays = [
  (final: prev: { sway-ahk = inputs.sway-ahk.packages.${final.system}.default; })
];
services.sway-ahk = {
  enable = true;
  configFile = "/home/trevor/Repos/sway-ahk/config/default.yaml";
};
```

Manage it with `systemctl --user start|stop|restart sway-ahk` (restart after config changes). The unit provides its own PATH (`swaymsg`, `pw-play`, `notify-send`), and logs still go to `/tmp/sway-ahk.log`. When running under systemd, prefer `systemctl` over the manual toggle command — the daemon writes the PID file itself, so the toggle will stop the service's daemon rather than start a second one.

Note: user services inherit the supplementary groups of the systemd **user manager**, which are fixed when it starts — not your login groups. If remaps don't trigger under the service after adding yourself to the `input` group, the user manager is stale: fully log out of all sessions, reboot, or `sudo systemctl restart user@1000.service` to pick up the group.
