// Package redact provides privacy filtering for sensitive data.
package redact

import "strings"

// DefaultHeaderDenylist contains headers that should be redacted by default.
var DefaultHeaderDenylist = []string{
	"cookie",
	"set-cookie",
	"authorization",
	"proxy-authorization",
	"x-api-key",
	"x-auth-token",
	"x-csrf-token",
	"x-xsrf-token",
}

// DefaultBodyFieldDenylist contains JSON field names that should be redacted by default.
var DefaultBodyFieldDenylist = []string{
	"password",
	"passwd",
	"secret",
	"token",
	"apikey",
	"api_key",
	"accesstoken",
	"access_token",
	"refreshtoken",
	"refresh_token",
	"private_key",
	"privatekey",
	"client_secret",
	"clientsecret",
	"credential",
	"credentials",
	"auth",
	"ssn",
	"social_security",
	"credit_card",
	"creditcard",
	"card_number",
	"cardnumber",
	"cvv",
	"pin",
}

// matchHeaderName checks if a header name matches a pattern (case-insensitive).
func matchHeaderName(actual, pattern string) bool {
	return strings.EqualFold(actual, pattern)
}

// matchBodyFieldName checks if a JSON field name matches a pattern (case-insensitive).
func matchBodyFieldName(actual, pattern string) bool {
	actualLower := strings.ToLower(actual)
	patternLower := strings.ToLower(pattern)

	// Exact match.
	if actualLower == patternLower {
		return true
	}

	// Check if the field name contains the pattern as a substring.
	// This catches variations like "user_password", "passwordHash", etc.
	return strings.Contains(actualLower, patternLower)
}
