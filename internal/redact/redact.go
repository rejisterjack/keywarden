package redact

import "regexp"

var sensitiveKeyPattern = regexp.MustCompile(`(?i)(authorization|password|passwd|secret|token|api[_-]?key|dsn|credential|private[_-]?key|nonce|ciphertext|session)`)

var (
	userinfo = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.-]*://)([^/\s:@]+):([^@\s]+)@`)
	bearer   = regexp.MustCompile(`(?i)bearer\s+\S+`)
	taskTok  = regexp.MustCompile(`\bkwt_[A-Za-z0-9_-]+\b`)
	skKey    = regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{8,}\b`)
)

// SensitiveKey reports whether a map or log key should be removed entirely.
func SensitiveKey(key string) bool {
	return sensitiveKeyPattern.MatchString(key)
}

// String masks secret-shaped substrings. It is safe to call more than once.
func String(s string) string {
	s = userinfo.ReplaceAllString(s, "${1}${2}:[redacted]@")
	s = bearer.ReplaceAllString(s, "Bearer [redacted]")
	s = taskTok.ReplaceAllString(s, "kwt_[redacted]")
	s = skKey.ReplaceAllString(s, "sk-[redacted]")
	return s
}

// Value returns a deep copy with secret-shaped keys and strings removed.
func Value(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if SensitiveKey(k) {
				out[k] = "[redacted]"
				continue
			}
			out[k] = Value(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = Value(val)
		}
		return out
	case string:
		return String(t)
	default:
		return v
	}
}
