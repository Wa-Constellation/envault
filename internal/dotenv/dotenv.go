package dotenv

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// ParseFile reads a dotenv file and returns key-value pairs.
func ParseFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening dotenv file: %w", err)
	}
	defer func() { _ = f.Close() }()
	return Parse(f)
}

// Parse reads dotenv-formatted content from r and returns key-value pairs.
// Supports:
//   - KEY=VALUE
//   - KEY="VALUE" (double-quoted, supports \n, \t, \" escapes)
//   - KEY='VALUE' (single-quoted, literal - no escapes)
//   - export KEY=VALUE (export prefix stripped)
//   - # comments (ignored)
//   - blank lines (ignored)
func Parse(r io.Reader) (map[string]string, error) {
	result := make(map[string]string)
	scanner := bufio.NewScanner(r)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Strip "export " prefix
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimPrefix(line, "export ")
			line = strings.TrimSpace(line)
		}

		// Find the = separator
		eqIdx := strings.Index(line, "=")
		if eqIdx == -1 {
			return nil, fmt.Errorf("line %d: missing '=' in %q", lineNum, line)
		}

		key := strings.TrimSpace(line[:eqIdx])
		value := strings.TrimSpace(line[eqIdx+1:])

		if key == "" {
			return nil, fmt.Errorf("line %d: empty key", lineNum)
		}

		// Parse the value based on quoting
		parsed, err := parseValue(value)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		result[key] = parsed
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading dotenv: %w", err)
	}

	return result, nil
}

// parseValue handles quoted and unquoted values.
func parseValue(s string) (string, error) {
	if len(s) == 0 {
		return "", nil
	}

	// Double-quoted value
	if s[0] == '"' {
		if len(s) < 2 || s[len(s)-1] != '"' {
			return "", fmt.Errorf("unterminated double quote in %q", s)
		}
		return unescapeDoubleQuoted(s[1 : len(s)-1])
	}

	// Single-quoted value (literal, no escapes)
	if s[0] == '\'' {
		if len(s) < 2 || s[len(s)-1] != '\'' {
			return "", fmt.Errorf("unterminated single quote in %q", s)
		}
		return s[1 : len(s)-1], nil
	}

	// Unquoted value: strip inline comments
	if idx := strings.Index(s, " #"); idx != -1 {
		s = strings.TrimSpace(s[:idx])
	}

	return s, nil
}

// unescapeDoubleQuoted processes escape sequences in double-quoted strings.
func unescapeDoubleQuoted(s string) (string, error) {
	var b strings.Builder
	b.Grow(len(s))

	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			default:
				// Keep the backslash for unknown escapes
				b.WriteByte('\\')
				b.WriteByte(s[i+1])
			}
			i++ // skip the next char
		} else {
			b.WriteByte(s[i])
		}
	}

	return b.String(), nil
}

// Quote returns a dotenv-safe representation of a value.
// If value contains spaces, newlines, quotes, or special characters,
// it is double-quoted with proper escapes.
func Quote(value string) string {
	// Check if quoting is needed
	needsQuoting := false
	for _, c := range value {
		if c == ' ' || c == '\n' || c == '\t' || c == '\r' || c == '"' || c == '\'' || c == '#' || c == '\\' {
			needsQuoting = true
			break
		}
	}

	if !needsQuoting && len(value) > 0 {
		return value
	}

	// Double-quote with escapes
	var b strings.Builder
	b.WriteByte('"')
	for _, c := range value {
		switch c {
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		default:
			b.WriteRune(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// Marshal writes vars as dotenv format to w, sorted by key.
func Marshal(w io.Writer, vars map[string]string) error {
	keys := sortedKeys(vars)
	for _, k := range keys {
		if _, err := fmt.Fprintf(w, "%s=%s\n", k, Quote(vars[k])); err != nil {
			return err
		}
	}
	return nil
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
