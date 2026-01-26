package redact

import (
	"testing"
)

func TestRedactHeaders(t *testing.T) {
	r := New(true)

	tests := []struct {
		name     string
		headers  map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "redacts cookie header",
			headers: map[string]interface{}{
				"Cookie":       "session=abc123",
				"Content-Type": "application/json",
			},
			expected: map[string]interface{}{
				"Cookie":       RedactedValue,
				"Content-Type": "application/json",
			},
		},
		{
			name: "redacts authorization header",
			headers: map[string]interface{}{
				"Authorization": "Bearer token123",
				"Accept":        "*/*",
			},
			expected: map[string]interface{}{
				"Authorization": RedactedValue,
				"Accept":        "*/*",
			},
		},
		{
			name: "case insensitive matching",
			headers: map[string]interface{}{
				"COOKIE":        "value",
				"authorization": "value",
				"X-API-KEY":     "value",
			},
			expected: map[string]interface{}{
				"COOKIE":        RedactedValue,
				"authorization": RedactedValue,
				"X-API-KEY":     RedactedValue,
			},
		},
		{
			name:     "handles nil headers",
			headers:  nil,
			expected: nil,
		},
		{
			name:     "handles empty headers",
			headers:  map[string]interface{}{},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.RedactHeaders(tt.headers)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			for key, expectedVal := range tt.expected {
				if result[key] != expectedVal {
					t.Errorf("header %s: expected %v, got %v", key, expectedVal, result[key])
				}
			}
		})
	}
}

func TestRedactHeadersDisabled(t *testing.T) {
	r := New(false)

	headers := map[string]interface{}{
		"Cookie":        "session=abc123",
		"Authorization": "Bearer token",
	}

	result := r.RedactHeaders(headers)

	// When disabled, headers should pass through unchanged.
	if result["Cookie"] != "session=abc123" {
		t.Errorf("expected cookie to pass through, got %v", result["Cookie"])
	}
	if result["Authorization"] != "Bearer token" {
		t.Errorf("expected authorization to pass through, got %v", result["Authorization"])
	}
}

func TestRedactBody(t *testing.T) {
	r := New(true)

	tests := []struct {
		name     string
		body     string
		contains string
		excludes string
	}{
		{
			name:     "redacts password field",
			body:     `{"username":"john","password":"secret123"}`,
			contains: RedactedValue,
			excludes: "secret123",
		},
		{
			name:     "redacts nested token field",
			body:     `{"user":{"name":"john","token":"abc123"}}`,
			contains: RedactedValue,
			excludes: "abc123",
		},
		{
			name:     "redacts api_key field",
			body:     `{"api_key":"key123","data":"value"}`,
			contains: RedactedValue,
			excludes: "key123",
		},
		{
			name:     "preserves non-sensitive fields",
			body:     `{"name":"john","email":"john@example.com"}`,
			contains: "john@example.com",
			excludes: "",
		},
		{
			name:     "handles non-JSON body",
			body:     "plain text body",
			contains: "plain text body",
			excludes: "",
		},
		{
			name:     "handles empty body",
			body:     "",
			contains: "",
			excludes: "",
		},
		{
			name:     "handles array with sensitive fields",
			body:     `[{"password":"pass1"},{"password":"pass2"}]`,
			contains: RedactedValue,
			excludes: "pass1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.RedactBody(tt.body)
			if tt.contains != "" && !containsString(result, tt.contains) {
				t.Errorf("expected result to contain %q, got %q", tt.contains, result)
			}
			if tt.excludes != "" && containsString(result, tt.excludes) {
				t.Errorf("expected result to not contain %q, got %q", tt.excludes, result)
			}
		})
	}
}

func TestRedactBodyDisabled(t *testing.T) {
	r := New(false)

	body := `{"password":"secret123"}`
	result := r.RedactBody(body)

	if !containsString(result, "secret123") {
		t.Errorf("expected password to pass through when disabled, got %s", result)
	}
}

func TestCustomRules(t *testing.T) {
	r := NewWithCustomRules(true, []string{"x-custom-header"}, []string{"custom_field"})

	// Test custom header.
	headers := map[string]interface{}{
		"X-Custom-Header": "value",
		"Other-Header":    "value",
	}
	result := r.RedactHeaders(headers)
	if result["X-Custom-Header"] != RedactedValue {
		t.Errorf("expected custom header to be redacted")
	}
	if result["Other-Header"] != "value" {
		t.Errorf("expected other header to pass through")
	}

	// Test custom body field.
	body := `{"custom_field":"secret","other_field":"public"}`
	bodyResult := r.RedactBody(body)
	if !containsString(bodyResult, RedactedValue) {
		t.Errorf("expected custom_field to be redacted")
	}
	if !containsString(bodyResult, "public") {
		t.Errorf("expected other_field to pass through")
	}
}

func TestMatchBodyFieldNameSubstring(t *testing.T) {
	r := New(true)

	// Test that fields containing sensitive patterns are redacted.
	body := `{"user_password":"secret","passwordHash":"hash123","mytoken":"abc"}`
	result := r.RedactBody(body)

	if containsString(result, "secret") {
		t.Errorf("expected user_password to be redacted")
	}
	if containsString(result, "hash123") {
		t.Errorf("expected passwordHash to be redacted")
	}
	if containsString(result, "abc") {
		t.Errorf("expected mytoken to be redacted")
	}
}

func containsString(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
