package domain

import (
	"fmt"
	"net/mail"
	"strings"
)

// NormalizeEmail trims and lower-cases an e-mail address.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// IsAllowedEmailDomain reports whether email's domain exactly equals (case-insensitive) one of the allowed domains.
func IsAllowedEmailDomain(email string, allowed []string) bool {
	email = NormalizeEmail(email)
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 {
		return false
	}
	domain := email[at+1:]
	for _, a := range allowed {
		if domain == strings.ToLower(strings.TrimSpace(a)) {
			return true
		}
	}
	return false
}

// ValidateEmail normalises email, checks its syntax and that its domain is allowed.
func ValidateEmail(email string, allowed []string) (string, error) {
	email = NormalizeEmail(email)
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", fmt.Errorf("%w: invalid email address", ErrInvalidInput)
	}
	if !IsAllowedEmailDomain(email, allowed) {
		return "", fmt.Errorf("%w: email domain is not allowed", ErrInvalidInput)
	}
	return email, nil
}
