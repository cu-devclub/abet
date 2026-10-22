package config

import "testing"

// D28: the dev act-as feature needs BOTH DEV_ACT_AS_ENABLED=true and APP_ENV=development; anything else keeps it OFF.
func TestDevActAsEnabled(t *testing.T) {
	cases := []struct {
		name      string
		requested bool
		appEnv    string
		want      bool
	}{
		{"requested in development", true, "development", true},
		{"requested in production", true, "production", false},
		{"requested with empty APP_ENV", true, "", false},
		{"not requested in development", false, "development", false},
		{"not requested in production", false, "production", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Config{DevActAsRequested: tc.requested, AppEnv: tc.appEnv}
			if got := c.DevActAsEnabled(); got != tc.want {
				t.Fatalf("DevActAsEnabled() = %v, want %v", got, tc.want)
			}
		})
	}
}
