package slug

import "regexp"

var re = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,62}$`)

// OK reports whether id is a safe agent, tool, or rule identifier.
// Identifiers cannot contain "__" because that separates tool id from tool name.
func OK(id string) bool {
	if !re.MatchString(id) {
		return false
	}
	for i := 0; i+1 < len(id); i++ {
		if id[i] == '_' && id[i+1] == '_' {
			return false
		}
	}
	return true
}
