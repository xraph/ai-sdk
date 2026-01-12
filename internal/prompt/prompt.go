// Package prompt provides shared prompt rendering utilities.
package prompt

import (
	"fmt"
	"strings"
)

// Render renders a prompt template with variables using {{.key}} placeholder syntax.
// Returns the rendered prompt with all placeholders replaced by their values.
func Render(template string, vars map[string]any) (string, error) {
	if len(vars) == 0 {
		return template, nil
	}

	result := template
	for key, value := range vars {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprint(value))
	}

	return result, nil
}
