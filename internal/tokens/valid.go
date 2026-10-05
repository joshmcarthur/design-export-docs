package tokens

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	nameRe     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)
	idRe       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)
	hexRe      = regexp.MustCompile(`^#([0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)
	colorFnRe  = regexp.MustCompile(`^(?i:rgb|rgba|hsl|hsla|oklch|oklab|lab|lch|color)\([0-9a-zA-Z .,%/+-]*\)$`)
	aliasRe    = regexp.MustCompile(`^\{([^{}]+)\}$`)
	lengthRe   = regexp.MustCompile(`^-?(\d+\.?\d*|\.\d+)(px|rem|em|%)?$`)
	plainCSSRe = regexp.MustCompile(`^[A-Za-z0-9 #%(),./+_-]+$`)
)

// cleanName strips a leading "--" or "." and validates the result.
func cleanName(raw string) (string, bool) {
	n := strings.TrimPrefix(raw, "--")
	n = strings.TrimPrefix(n, ".")
	return n, nameRe.MatchString(n)
}

// cssIdent escapes a validated name for use as a custom property or class name.
func cssIdent(n string) string { return strings.ReplaceAll(n, ".", `\.`) }

func balanced(s string) bool {
	depth := 0
	for _, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0
}

// colorValue validates a colour value and returns its normalised form. Aliases are returned
// as "{name}" and resolved later.
func colorValue(v any) (string, bool) {
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	switch {
	case hexRe.MatchString(s):
		return strings.ToLower(s), true
	case colorFnRe.MatchString(s):
		return s, true
	case aliasRe.MatchString(s):
		return s, true
	}
	return "", false
}

// shadowValue validates a box-shadow string.
func shadowValue(v any) (string, bool) {
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	low := strings.ToLower(s)
	if s == "" || len(s) > 400 || !plainCSSRe.MatchString(s) || !balanced(s) ||
		strings.Contains(low, "var(") || strings.Contains(low, "url(") {
		return "", false
	}
	return s, true
}

// plainValue validates a single CSS value of an "other" family.
func plainValue(v any) (string, bool) {
	var s string
	switch x := v.(type) {
	case string:
		s = strings.TrimSpace(x)
	case float64:
		s = formatNumber(x)
	default:
		return "", false
	}
	low := strings.ToLower(s)
	if s == "" || len(s) > 200 || !plainCSSRe.MatchString(s) || !balanced(s) ||
		strings.Contains(low, "var(") || strings.Contains(low, "url(") {
		return "", false
	}
	return s, true
}

// lengthValue validates a length: a number (px) or a value with px, rem, em or %.
func lengthValue(v any) (string, bool) {
	switch x := v.(type) {
	case float64:
		if x == 0 {
			return "0", true
		}
		return formatNumber(x) + "px", true
	case string:
		s := strings.TrimSpace(x)
		if !lengthRe.MatchString(s) {
			return "", false
		}
		if _, err := strconv.ParseFloat(s, 64); err == nil { // bare number string
			f, _ := strconv.ParseFloat(s, 64)
			if f == 0 {
				return "0", true
			}
			return s + "px", true
		}
		return s, true
	}
	return "", false
}

func lineHeightValue(v any) (string, bool) {
	if f, ok := v.(float64); ok && f < 10 {
		return formatNumber(f), true
	}
	if s, ok := v.(string); ok {
		if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil && f < 10 {
			return strings.TrimSpace(s), true
		}
	}
	return lengthValue(v)
}

func fontWeightValue(v any) (string, bool) {
	switch x := v.(type) {
	case float64:
		if x >= 1 && x <= 1000 {
			return formatNumber(x), true
		}
	case string:
		s := strings.TrimSpace(x)
		switch s {
		case "normal":
			return "400", true
		case "bold":
			return "700", true
		}
		parts := strings.Fields(s)
		if len(parts) < 1 || len(parts) > 2 {
			return "", false
		}
		for _, p := range parts {
			f, err := strconv.ParseFloat(p, 64)
			if err != nil || f < 1 || f > 1000 {
				return "", false
			}
		}
		return strings.Join(parts, " "), true
	}
	return "", false
}

// familyStack validates a font-family stack.
func familyStack(s string) bool {
	if s == "" || len(s) > 200 || strings.ContainsAny(s, ";{}<>\\()") {
		return false
	}
	return strings.Count(s, `"`)%2 == 0 && strings.Count(s, "'")%2 == 0
}

func formatNumber(f float64) string {
	if f == float64(int64(f)) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func warnf(w *[]string, format string, args ...any) {
	*w = append(*w, fmt.Sprintf(format, args...))
}
