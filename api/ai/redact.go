package ai

import (
	"net/url"
	"strings"
)

const redactedValue = "[REDACTED]"

var sensitiveKeyPatterns = []string{
	"PASSWORD",
	"PASSWD",
	"SECRET",
	"TOKEN",
	"API_KEY",
	"APIKEY",
	"PRIVATE",
	"CREDENTIAL",
	"AUTH",
	"KEY",
}

// isSensitiveKey reports whether a key name looks like it may hold a secret.
func isSensitiveKey(key string) bool {
	upper := strings.ToUpper(key)
	for _, pattern := range sensitiveKeyPatterns {
		if strings.Contains(upper, pattern) {
			return true
		}
	}
	return false
}

// redactEnv takes a slice of "KEY=VALUE" strings and replaces the values of
// sensitive keys with a redaction marker. Non-sensitive entries are kept as-is.
func redactEnv(env []string) []string {
	if len(env) == 0 {
		return nil
	}

	out := make([]string, 0, len(env))
	for _, e := range env {
		idx := strings.IndexByte(e, '=')
		if idx < 0 {
			out = append(out, redactedValue)
			continue
		}
		key := e[:idx]
		if isSensitiveKey(key) {
			out = append(out, key+"="+redactedValue)
		} else {
			out = append(out, e)
		}
	}
	return out
}

// sanitizeURL strips any embedded credentials (user:pass@) from a URL so that
// secrets are not leaked to the LLM.
func sanitizeURL(raw string) string {
	if raw == "" {
		return raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	if u.User != nil {
		u.User = nil
	}

	return u.String()
}

// redactLabels redacts label values whose keys look sensitive.
func redactLabels(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}

	out := make(map[string]string, len(labels))
	for k, v := range labels {
		if isSensitiveKey(k) {
			out[k] = redactedValue
		} else {
			out[k] = v
		}
	}
	return out
}
