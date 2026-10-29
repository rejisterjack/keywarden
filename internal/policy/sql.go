package policy

import (
	"regexp"
	"strings"
)

// Classify labels one SQL statement.
// destructive: DDL, admin commands, and multi-statement batches.
// mutating: INSERT, UPDATE, DELETE, MERGE.
// read: SELECT, WITH, SHOW, EXPLAIN, VALUES, TABLE.
// Anything unrecognized is destructive so an allow-read rule does not catch it.
func Classify(sql string) string {
	masked := maskLiterals(sql)
	if strings.TrimSpace(masked) == "" {
		return "destructive"
	}
	if statementCount(masked) != 1 {
		return "destructive"
	}
	upper := strings.ToUpper(masked)
	if containsWord(upper, destructiveWords) {
		return "destructive"
	}
	if selectInto.MatchString(upper) && !hasWord(upper, "INSERT") {
		return "destructive"
	}
	switch firstWord(upper) {
	case "SET", "RESET", "DO":
		// These are destructive as statements. SET also appears inside UPDATE,
		// and DO appears inside ON CONFLICT, so they are not matched anywhere.
		return "destructive"
	}
	if containsWord(upper, mutatingWords) {
		return "mutating"
	}
	first := firstWord(upper)
	switch first {
	case "SELECT", "WITH", "SHOW", "EXPLAIN", "VALUES", "TABLE":
		return "read"
	default:
		return "destructive"
	}
}

var destructiveWords = []string{
	"DROP", "TRUNCATE", "ALTER", "GRANT", "REVOKE", "CREATE", "COMMENT",
	"VACUUM", "COPY", "CALL", "EXECUTE", "PREPARE", "DEALLOCATE",
	"DISCARD", "LOCK", "REASSIGN", "IMPORT", "LISTEN",
	"UNLISTEN", "NOTIFY", "CHECKPOINT", "CLUSTER", "REINDEX", "REFRESH",
}

var mutatingWords = []string{"INSERT", "UPDATE", "DELETE", "MERGE"}

var selectInto = regexp.MustCompile(`\bSELECT\b[\s\S]*\bINTO\b`)

func containsWord(sql string, words []string) bool {
	for _, w := range words {
		if hasWord(sql, w) {
			return true
		}
	}
	return false
}

func hasWord(sql, w string) bool {
	rest := sql
	for {
		i := strings.Index(rest, w)
		if i < 0 {
			return false
		}
		before := i == 0 || !isWordByte(rest[i-1])
		end := i + len(w)
		after := end == len(rest) || !isWordByte(rest[end])
		if before && after {
			return true
		}
		rest = rest[i+1:]
	}
}

func isWordByte(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '_'
}

func firstWord(sql string) string {
	fields := strings.Fields(sql)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// Statements counts SQL statements after comments and literals are masked.
// A trailing semicolon does not count as a second statement.
func Statements(sql string) int {
	return statementCount(maskLiterals(sql))
}

func statementCount(masked string) int {
	n := 0
	for _, part := range strings.Split(masked, ";") {
		if strings.TrimSpace(part) != "" {
			n++
		}
	}
	return n
}

func maskLiterals(sql string) string {
	var b strings.Builder
	b.Grow(len(sql))
	i := 0
	for i < len(sql) {
		switch {
		case strings.HasPrefix(sql[i:], "--"):
			i = maskUntil(sql, i, &b, func(s string, j int) (int, bool) {
				if s[j] == '\n' {
					return j, true
				}
				return j + 1, false
			})
		case strings.HasPrefix(sql[i:], "/*"):
			b.WriteString("  ")
			i += 2
			for i < len(sql) {
				if strings.HasPrefix(sql[i:], "*/") {
					b.WriteString("  ")
					i += 2
					break
				}
				if sql[i] == '\n' {
					b.WriteByte('\n')
				} else {
					b.WriteByte(' ')
				}
				i++
			}
		case sql[i] == '\'':
			i = maskQuoted(sql, i, &b, '\'')
		case sql[i] == '"':
			i = maskQuoted(sql, i, &b, '"')
		case sql[i] == '$':
			if tag, ok := dollarTag(sql[i:]); ok {
				closer := "$" + tag + "$"
				b.WriteString(strings.Repeat(" ", len(closer)))
				i += len(closer)
				for i < len(sql) && !strings.HasPrefix(sql[i:], closer) {
					if sql[i] == '\n' {
						b.WriteByte('\n')
					} else {
						b.WriteByte(' ')
					}
					i++
				}
				if strings.HasPrefix(sql[i:], closer) {
					b.WriteString(strings.Repeat(" ", len(closer)))
					i += len(closer)
				}
				continue
			}
			b.WriteByte(sql[i])
			i++
		default:
			b.WriteByte(sql[i])
			i++
		}
	}
	return b.String()
}

func maskUntil(sql string, i int, b *strings.Builder, stop func(string, int) (int, bool)) int {
	for i < len(sql) {
		next, done := stop(sql, i)
		for j := i; j < next && j < len(sql); j++ {
			if sql[j] == '\n' {
				b.WriteByte('\n')
			} else {
				b.WriteByte(' ')
			}
		}
		i = next
		if done {
			return i
		}
	}
	return i
}

func maskQuoted(sql string, i int, b *strings.Builder, quote byte) int {
	b.WriteByte(' ')
	i++
	for i < len(sql) {
		if sql[i] == quote {
			if i+1 < len(sql) && sql[i+1] == quote {
				b.WriteString("  ")
				i += 2
				continue
			}
			b.WriteByte(' ')
			return i + 1
		}
		if sql[i] == '\n' {
			b.WriteByte('\n')
		} else {
			b.WriteByte(' ')
		}
		i++
	}
	return i
}

func dollarTag(s string) (string, bool) {
	if len(s) < 2 || s[0] != '$' {
		return "", false
	}
	j := 1
	for j < len(s) && isIdent(s[j]) {
		j++
	}
	if j < len(s) && s[j] == '$' {
		return s[1:j], true
	}
	return "", false
}

func isIdent(b byte) bool {
	return b == '_' || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}
