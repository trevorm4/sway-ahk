package compositor

import (
	"testing"
)

func TestParseSwayFocusEvent(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
		ok   bool
	}{
		{
			name: "app id",
			data: `{"change":"focus","container":{"app_id":"firefox","window_properties":{}}}`,
			want: "firefox",
			ok:   true,
		},
		{
			name: "x11 class fallback",
			data: `{"change":"focus","container":{"app_id":null,"window_properties":{"class":"steam_app_1974050"}}}`,
			want: "steam_app_1974050",
			ok:   true,
		},
		{
			name: "non-focus event ignored",
			data: `{"change":"title","container":{"app_id":"firefox"}}`,
			want: "",
			ok:   false,
		},
		{
			name: "no identifier ignored",
			data: `{"change":"focus","container":{"app_id":null,"window_properties":{}}}`,
			want: "",
			ok:   false,
		},
		{
			name: "garbage ignored",
			data: `not json`,
			want: "",
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseSwayFocusEvent([]byte(tt.data))
			if ok != tt.ok || got != tt.want {
				t.Errorf("parseSwayFocusEvent(%q) = (%q, %v), want (%q, %v)", tt.data, got, ok, tt.want, tt.ok)
			}
		})
	}
}
