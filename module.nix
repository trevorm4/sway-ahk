{
  config,
  lib,
  pkgs,
  ...
}:
let
  cfg = config.services.sway-ahk;
in
{
  options.services.sway-ahk = {
    enable = lib.mkEnableOption "sway-ahk key automation daemon";

    package = lib.mkOption {
      type = lib.types.package;
      default = pkgs.sway-ahk;
      defaultText = lib.literalExpression "pkgs.sway-ahk";
      description = "The sway-ahk package to use.";
    };

    configFile = lib.mkOption {
      type = lib.types.path;
      description = "Path to the sway-ahk YAML configuration file.";
    };

    compositor = lib.mkOption {
      type = lib.types.enum [
        "sway"
        "hyprland"
      ];
      default = "sway";
      description = "Which compositor integration to use; provides the matching CLI tool (swaymsg or hyprctl) in PATH.";
    };
  };

  config = lib.mkIf cfg.enable {
    systemd.user.services.sway-ahk = {
      description = "sway-ahk: key automation for Sway and Hyprland";
      wantedBy = [ "graphical-session.target" ];
      partOf = [ "graphical-session.target" ];
      after = [ "graphical-session.target" ];
      serviceConfig = {
        ExecStart = "${cfg.package}/bin/sway-ahk -config ${toString cfg.configFile} -compositor ${cfg.compositor}";
        Environment = [
          "SWAY_AHK_DAEMON=1"
          "PATH=${lib.makeBinPath [
            (if cfg.compositor == "hyprland" then pkgs.hyprland else pkgs.sway)
            pkgs.pipewire
            pkgs.libnotify
          ]}"
        ];
        Restart = "on-failure";
        RestartSec = "5s";
      };
    };
  };
}
