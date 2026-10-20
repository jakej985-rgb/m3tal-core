package plugin

import (
	"regexp"
)

// Parameterize replaces occurrences of ${VAR} or $VAR in the template
// with values from the vars map. If a variable is missing, it is left empty or replaced by "".
func Parameterize(template string, vars map[string]string) string {
	// Match ${VAR} or $VAR
	// We'll use a regex that matches ${VAR} and extracts the group VAR.
	re := regexp.MustCompile(`\$\{([A-Za-z0-9_]+)\}`)
	return re.ReplaceAllStringFunc(template, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		key := submatches[1]
		if val, ok := vars[key]; ok {
			return val
		}
		return ""
	})
}
