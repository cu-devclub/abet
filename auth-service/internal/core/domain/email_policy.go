package domain

import (
	"fmt"
	"strings"
)

// EmailDomainPolicy accepts emails whose domain EXACTLY equals one of the allowed domains
// (case-insensitive, no suffix / subdomain matching).
type EmailDomainPolicy struct {
	domains []string
}

// NewEmailDomainPolicy builds a policy from a list of domains; at least one is required.
func NewEmailDomainPolicy(domains []string) (*EmailDomainPolicy, error) {
	cleaned := make([]string, 0, len(domains))
	for _, d := range domains {
		d = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(d), "@")))
		if d != "" {
			cleaned = append(cleaned, d)
		}
	}
	if len(cleaned) == 0 {
		return nil, fmt.Errorf("%w: at least one allowed email domain is required", ErrInvalidInput)
	}
	return &EmailDomainPolicy{domains: cleaned}, nil
}

// Allows reports whether the email's domain is in the allowed list.
func (p *EmailDomainPolicy) Allows(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 {
		return false
	}
	domain := email[at+1:]
	for _, d := range p.domains {
		if domain == d {
			return true
		}
	}
	return false
}

// PrimaryDomain returns the first allowed domain (used as Google's hd hint).
func (p *EmailDomainPolicy) PrimaryDomain() string {
	return p.domains[0]
}
