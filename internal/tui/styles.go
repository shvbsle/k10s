package tui

// stripAnsi removes ANSI escape sequences for accurate length calculation.
func stripAnsi(s string) string {
	result := ""
	inEscape := false
	escapeStart := false

	for i := 0; i < len(s); i++ {
		r := rune(s[i])

		if r == '\x1b' {
			inEscape = true
			escapeStart = true
			continue
		}

		if inEscape {
			if escapeStart && r == '[' {
				escapeStart = false
				continue
			}
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
				escapeStart = false
				continue
			}
			continue
		}

		if !inEscape {
			result += string(r)
		}
	}
	return result
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
