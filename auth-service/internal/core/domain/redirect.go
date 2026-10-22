package domain

import "strings"

// SanitizeRedirect returns redirect when it is a safe same-origin relative path,
// otherwise "/". A safe path starts with a single "/" and contains no backslash or control characters.
func SanitizeRedirect(redirect string) string {
	if !strings.HasPrefix(redirect, "/") || strings.HasPrefix(redirect, "//") {
		return "/"
	}
	for _, r := range redirect {
		if r == '\\' || r < 0x20 || r == 0x7f {
			return "/"
		}
	}
	return redirect
}
