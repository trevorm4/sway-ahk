package compositor

import (
	"testing"
)

func TestParseHyprlandFocusEvent(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
		ok   bool
	}{
		{"active window", "activewindow>>firefox,GitHub - Mozilla Firefox\n", "firefox", true},
		{"no trailing newline", "activewindow>>steam_app_1974050,Some Game", "steam_app_1974050", true},
		{"class with comma in title only", "activewindow>>emacs,a, b, c\n", "emacs", true},
		{"empty workspace ignored", "activewindow>>,\n", "", false},
		{"activewindowv2 ignored", "activewindowv2>>5a1b2c3d4e5f\n", "", false},
		{"other event ignored", "workspace>>2\n", "", false},
		{"close window ignored", "closewindow>>5a1b2c3d4e5f\n", "", false},
		{"garbage ignored", "garbage\n", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseHyprlandFocusEvent(tt.line)
			if ok != tt.ok || got != tt.want {
				t.Errorf("parseHyprlandFocusEvent(%q) = (%q, %v), want (%q, %v)", tt.line, got, ok, tt.want, tt.ok)
			}
		})
	}
}
