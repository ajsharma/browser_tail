// Package redact provides privacy filtering for sensitive data.
package redact

import (
	"encoding/json"
)

// RedactedValue is the placeholder for redacted content.
const RedactedValue = "[REDACTED]"

// Redactor handles redaction of sensitive data.
type Redactor struct {
	enabled           bool
	headerDenylist    []string
	bodyFieldDenylist []string
}

// New creates a new Redactor with default settings.
func New(enabled bool) *Redactor {
	return &Redactor{
		enabled:           enabled,
		headerDenylist:    DefaultHeaderDenylist,
		bodyFieldDenylist: DefaultBodyFieldDenylist,
	}
}

// NewWithCustomRules creates a Redactor with custom denylist patterns.
func NewWithCustomRules(enabled bool, headers, bodyFields []string) *Redactor {
	r := New(enabled)
	if headers != nil {
		r.headerDenylist = append(r.headerDenylist, headers...)
	}
	if bodyFields != nil {
		r.bodyFieldDenylist = append(r.bodyFieldDenylist, bodyFields...)
	}
	return r
}

// IsEnabled returns whether redaction is enabled.
func (r *Redactor) IsEnabled() bool {
	return r.enabled
}

// RedactHeaders redacts sensitive headers from a header map.
func (r *Redactor) RedactHeaders(headers map[string]interface{}) map[string]interface{} {
	if !r.enabled || headers == nil {
		return headers
	}

	result := make(map[string]interface{}, len(headers))
	for key, value := range headers {
		if r.shouldRedactHeader(key) {
			result[key] = RedactedValue
		} else {
			result[key] = value
		}
	}
	return result
}

// RedactBody redacts sensitive fields from a response body.
// It attempts to parse the body as JSON and redact matching fields.
// If parsing fails, the body is returned unchanged.
func (r *Redactor) RedactBody(body string) string {
	if !r.enabled || body == "" {
		return body
	}

	// Try to parse as JSON.
	var data interface{}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		// Not JSON, return as-is.
		return body
	}

	// Redact the parsed data.
	redacted := r.redactValue(data)

	// Re-encode to JSON.
	result, err := json.Marshal(redacted)
	if err != nil {
		return body
	}

	return string(result)
}

// shouldRedactHeader checks if a header should be redacted.
func (r *Redactor) shouldRedactHeader(name string) bool {
	for _, pattern := range r.headerDenylist {
		if matchHeaderName(name, pattern) {
			return true
		}
	}
	return false
}

// shouldRedactBodyField checks if a body field should be redacted.
func (r *Redactor) shouldRedactBodyField(name string) bool {
	for _, pattern := range r.bodyFieldDenylist {
		if matchBodyFieldName(name, pattern) {
			return true
		}
	}
	return false
}

// redactValue recursively redacts sensitive fields in a JSON value.
func (r *Redactor) redactValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return r.redactMap(val)
	case []interface{}:
		return r.redactSlice(val)
	default:
		return val
	}
}

// redactMap redacts sensitive fields in a JSON object.
func (r *Redactor) redactMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for key, value := range m {
		if r.shouldRedactBodyField(key) {
			result[key] = RedactedValue
		} else {
			result[key] = r.redactValue(value)
		}
	}
	return result
}

// redactSlice redacts sensitive fields in a JSON array.
func (r *Redactor) redactSlice(s []interface{}) []interface{} {
	result := make([]interface{}, len(s))
	for i, value := range s {
		result[i] = r.redactValue(value)
	}
	return result
}
